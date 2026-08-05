package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Migrations is the GORM model backing the "migrations" tracking table -
// one row per applied migration file, per batch.
type Migrations struct {
	ID        uint      `gorm:"primaryKey"`
	Migration string    `gorm:"type:varchar(255);not null;unique"`
	Batch     int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// MigrationContent is a migration file's Up/Down SQL, split out by
// ParseMigrationFile.
type MigrationContent struct {
	Up   string
	Down string
}

// MigrationOptions configures a MigrationManager run - which migration(s),
// how (force/pretend/step), and what to do around it (seed, drop views).
type MigrationOptions struct {
	Force     bool
	Path      string
	File      string
	Database  string
	Pretend   bool
	Step      int
	Create    string
	Table     string
	Seed      bool
	DropViews bool
	DropTypes bool
	RealPath  bool
	FullPath  bool
	Isolation string // "transaction", "none"
}

// MigrationManager runs, rolls back, and generates SQL migration files for
// one database connection. Build one with NewMigrationManager, then
// InitializeDatabase before calling anything that touches the database.
type MigrationManager struct {
	dm      *DatabaseManager
	config  *Config
	dbType  DatabaseType
	gormDB  *gorm.DB
	sqlDB   *sql.DB
	Options MigrationOptions
}

// NewMigrationManager builds a MigrationManager with transaction-based
// migration isolation by default (see MigrationOptions.Isolation).
func NewMigrationManager(dm *DatabaseManager, config *Config) *MigrationManager {
	return &MigrationManager{
		dm:     dm,
		config: config,
		Options: MigrationOptions{
			Isolation: "transaction", // Default to transaction-based
		},
	}
}

// SetOptions replaces the manager's MigrationOptions wholesale. The
// SetXxx methods below are shorthand for changing one field at a time.
func (mm *MigrationManager) SetOptions(options MigrationOptions) *MigrationManager {
	mm.Options = options
	return mm
}

// SetDatabaseType pins the engine this manager operates against.
func (mm *MigrationManager) SetDatabaseType(dbType DatabaseType) *MigrationManager {
	mm.dbType = dbType
	return mm
}

// SetForce sets whether destructive operations skip their confirmation
// prompt.
func (mm *MigrationManager) SetForce(force bool) *MigrationManager {
	mm.Options.Force = force
	return mm
}

// SetPath overrides the migration file directory (see GetMigrationPath).
func (mm *MigrationManager) SetPath(path string) *MigrationManager {
	mm.Options.Path = path
	return mm
}

// SetFile restricts a run to migration files whose name contains this
// substring.
func (mm *MigrationManager) SetFile(file string) *MigrationManager {
	mm.Options.File = file
	return mm
}

// SetDatabaseName sets which database InitializeDatabase connects to.
func (mm *MigrationManager) SetDatabaseName(database string) *MigrationManager {
	mm.Options.Database = database
	return mm
}

// SetPretend sets whether RunMigrationFile prints SQL instead of
// executing it.
func (mm *MigrationManager) SetPretend(pretend bool) *MigrationManager {
	mm.Options.Pretend = pretend
	return mm
}

// SetStep sets the batch-count limit used by Migrate/Rollback/Refresh.
func (mm *MigrationManager) SetStep(step int) *MigrationManager {
	mm.Options.Step = step
	return mm
}

// SetCreateTable sets the table name MakeMigration should scaffold a
// CREATE TABLE for.
func (mm *MigrationManager) SetCreateTable(table string) *MigrationManager {
	mm.Options.Create = table
	return mm
}

// SetAlterTable sets the table name MakeMigration should scaffold an
// ALTER TABLE stub for.
func (mm *MigrationManager) SetAlterTable(table string) *MigrationManager {
	mm.Options.Table = table
	return mm
}

// SetSeed sets whether seeders run automatically after Refresh/Fresh.
func (mm *MigrationManager) SetSeed(seed bool) *MigrationManager {
	mm.Options.Seed = seed
	return mm
}

// SetDropViews sets whether Fresh drops all views before recreating the
// database.
func (mm *MigrationManager) SetDropViews(dropViews bool) *MigrationManager {
	mm.Options.DropViews = dropViews
	return mm
}

// SetDropTypes sets whether Fresh drops custom types before recreating
// the database (PostgreSQL only).
func (mm *MigrationManager) SetDropTypes(dropTypes bool) *MigrationManager {
	mm.Options.DropTypes = dropTypes
	return mm
}

// SetRealPath sets whether SetPath's value is used as-is (true) or
// filepath.Clean'd first (false).
func (mm *MigrationManager) SetRealPath(realPath bool) *MigrationManager {
	mm.Options.RealPath = realPath
	return mm
}

// SetFullPath sets whether MakeMigration prints the created file's path.
func (mm *MigrationManager) SetFullPath(fullPath bool) *MigrationManager {
	mm.Options.FullPath = fullPath
	return mm
}

// SetIsolation sets how RunMigrationFile applies a migration: "transaction"
// wraps the SQL and the tracking-table insert in one DB transaction; any
// other value runs them as separate statements.
func (mm *MigrationManager) SetIsolation(isolation string) *MigrationManager {
	mm.Options.Isolation = isolation
	return mm
}

// SetConfig replaces the Config this manager reads connection/path
// settings from.
func (mm *MigrationManager) SetConfig(config *Config) *MigrationManager {
	mm.config = config
	return mm
}

// SetGormDB injects an already-open GORM connection instead of one from
// InitializeDatabase.
func (mm *MigrationManager) SetGormDB(gormDB *gorm.DB) *MigrationManager {
	mm.gormDB = gormDB
	return mm
}

// SetSqlDB injects an already-open *sql.DB instead of one from
// InitializeDatabase.
func (mm *MigrationManager) SetSqlDB(sqlDB *sql.DB) *MigrationManager {
	mm.sqlDB = sqlDB
	return mm
}

// InitializeDatabase opens the connection this manager operates on -
// against Options.Database if set, otherwise the engine's default/system
// connection.
func (mm *MigrationManager) InitializeDatabase(dbType DatabaseType) error {
	var err error
	mm.dbType = dbType

	if mm.Options.Database != "" {
		mm.gormDB, mm.sqlDB, err = mm.dm.InitializeDatabaseWithName(dbType, mm.Options.Database, mm.config)
	} else {
		mm.gormDB, mm.sqlDB, err = mm.dm.InitializeDatabase(dbType, mm.config)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	return nil
}

// GetGormDB returns the manager's active GORM connection.
func (mm *MigrationManager) GetGormDB() *gorm.DB {
	return mm.gormDB
}

// GetMigrationPath resolves where migration files live: Options.Path if
// set (cleaned unless RealPath), otherwise Config.MySQL.MigrationPath or
// Config.PostgreSQL.MigrationPath joined with "mysql"/"postgresql".
func (mm *MigrationManager) GetMigrationPath() string {
	if mm.Options.Path != "" {
		if mm.Options.RealPath {
			return mm.Options.Path
		}
		return filepath.Clean(mm.Options.Path)
	}

	switch mm.dbType {
	case MySQL:
		return filepath.Join(mm.config.MySQL.MigrationPath, "mysql")
	case PostgreSQL:
		return filepath.Join(mm.config.PostgreSQL.MigrationPath, "postgresql")
	default:
		return "./migrations"
	}
}

// InitMigrationTable creates the target database (if missing) and the
// "migrations" tracking table (if missing) - safe to call repeatedly.
func (mm *MigrationManager) InitMigrationTable() error {
	if err := mm.createDatabaseIfNotExists(); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	db := mm.gormDB.Session(&gorm.Session{PrepareStmt: false})
	if err := db.AutoMigrate(&Migrations{}); err != nil {
		if mm.dbType == PostgreSQL {
			err := mm.gormDB.Exec(`
                CREATE TABLE IF NOT EXISTS migrations (
                    id SERIAL PRIMARY KEY,
                    migration VARCHAR(255) NOT NULL,
                    batch INTEGER NOT NULL,
                    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
            `).Error
			if err != nil {
				return fmt.Errorf("failed to create migrations table: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

func (mm *MigrationManager) createDatabaseIfNotExists() error {
	var dbName string
	var createQuery string

	switch mm.dbType {
	case MySQL:
		dbName = mm.config.MySQL.DatabaseName
		charset := mm.config.MySQL.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		createQuery = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s", dbName, charset)
	case PostgreSQL:
		dbName = mm.config.PostgreSQL.DatabaseName
		createQuery = fmt.Sprintf(`CREATE DATABASE "%s" WITH ENCODING = 'UTF8'`, dbName)
	default:
		return fmt.Errorf("unsupported database type: %s", mm.dbType)
	}

	var exists bool
	switch mm.dbType {
	case MySQL:
		var count int64
		err := mm.gormDB.Raw("SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", dbName).Scan(&count).Error
		if err != nil {
			return err
		}
		exists = count > 0
	case PostgreSQL:
		var count int64
		err := mm.gormDB.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", dbName).Scan(&count).Error
		if err != nil {
			return err
		}
		exists = count > 0
	}

	if !exists {
		if err := mm.gormDB.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
	}

	return nil
}

// GetCurrentMigrations returns the names of every migration recorded as
// applied, oldest first.
func (mm *MigrationManager) GetCurrentMigrations() ([]string, error) {
	if err := mm.InitMigrationTable(); err != nil {
		return nil, err
	}

	var migrations []string
	if err := mm.gormDB.Model(&Migrations{}).
		Select("migration").
		Order("id ASC").
		Pluck("migration", &migrations).Error; err != nil {
		return nil, fmt.Errorf("failed to get current migrations: %w", err)
	}

	return migrations, nil
}

// GetNextBatch returns the batch number the next Migrate call would use
// (current max + 1).
func (mm *MigrationManager) GetNextBatch() (int, error) {
	if err := mm.InitMigrationTable(); err != nil {
		return 0, err
	}

	var lastBatch int
	err := mm.gormDB.Model(&Migrations{}).
		Select("COALESCE(MAX(batch), 0)").
		Scan(&lastBatch).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get next batch: %w", err)
	}

	return lastBatch + 1, nil
}

// GetLastBatch returns the highest batch number applied so far, or 0 if
// nothing has run yet.
func (mm *MigrationManager) GetLastBatch() (int, error) {
	if err := mm.InitMigrationTable(); err != nil {
		return 0, err
	}

	var lastBatch int
	err := mm.gormDB.Model(&Migrations{}).
		Select("COALESCE(MAX(batch), 0)").
		Scan(&lastBatch).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get last batch: %w", err)
	}

	return lastBatch, nil
}

// RecordMigration inserts (action "up") or deletes (action "down") the
// tracking row for migrationName.
func (mm *MigrationManager) RecordMigration(migrationName string, batch int, action string) error {
	switch action {
	case "up":
		migration := Migrations{
			Migration: migrationName,
			Batch:     batch,
			CreatedAt: time.Now(),
		}

		if err := mm.gormDB.Save(&migration).Error; err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}
	case "down":
		if err := mm.gormDB.Where("migration = ?", migrationName).Delete(&Migrations{}).Error; err != nil {
			return fmt.Errorf("failed to remove migration record: %w", err)
		}
	}

	return nil
}

// CleanOrphanedMigrations removes tracking rows whose migration file no
// longer exists on disk.
func (mm *MigrationManager) CleanOrphanedMigrations() error {
	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return err
	}

	migrationPath := mm.GetMigrationPath()
	cleanedCount := 0

	for _, migration := range currentMigrations {
		filePath := filepath.Join(migrationPath, migration+".sql")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := mm.gormDB.Where("migration = ?", migration).Delete(&Migrations{}).Error; err != nil {
				return fmt.Errorf("failed to remove orphaned migration: %w", err)
			}
			cleanedCount++
		}
	}

	return nil
}

// ParseMigrationFile splits a migration SQL file into its "-- +goose Up"
// and "-- +goose Down" sections.
func (mm *MigrationManager) ParseMigrationFile(filePath string) (*MigrationContent, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	fileContent := string(content)

	upRegex := regexp.MustCompile(`(?s)-- \+goose Up\s*\n(.*?)(?:-- \+goose Down|$)`)
	upMatch := upRegex.FindStringSubmatch(fileContent)
	var upContent string
	if len(upMatch) > 1 {
		upContent = strings.TrimSpace(upMatch[1])
	}

	downRegex := regexp.MustCompile(`(?s)-- \+goose Down\s*\n(.*?)(?:-- \+goose Up|$)`)
	downMatch := downRegex.FindStringSubmatch(fileContent)
	var downContent string
	if len(downMatch) > 1 {
		downContent = strings.TrimSpace(downMatch[1])
	}

	return &MigrationContent{
		Up:   upContent,
		Down: downContent,
	}, nil
}

// RunMigrationFile executes one migration file's up or down SQL and
// records (or removes) its tracking row - unless Options.Pretend is set,
// in which case it only prints the SQL. PostgreSQL "down" runs rewrite
// bare DROP TABLE to DROP TABLE CASCADE.
func (mm *MigrationManager) RunMigrationFile(filePath, action string, batch int) error {
	filename := filepath.Base(filePath)
	migrationName := strings.TrimSuffix(filename, ".sql")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("migration file not found: %s", filePath)
	}

	migrationContent, err := mm.ParseMigrationFile(filePath)
	if err != nil {
		return err
	}

	var sqlContent string
	switch action {
	case "up":
		sqlContent = migrationContent.Up
	case "down":
		if mm.dbType == PostgreSQL {
			sqlContent = strings.ReplaceAll(sqlContent, "DROP TABLE", "DROP TABLE CASCADE")
		}
	}

	if strings.TrimSpace(sqlContent) == "" {
		return nil
	}

	if mm.Options.Pretend {
		fmt.Printf("-- Pretending to run %s migration: %s\n", action, migrationName)
		fmt.Println(sqlContent)
		return nil
	}

	var execFunc func(string) error
	if mm.Options.Isolation == "transaction" {
		execFunc = func(sql string) error {
			tx := mm.gormDB.Begin()
			if tx.Error != nil {
				return tx.Error
			}

			if err := tx.Exec(sql).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("migration execution failed: %w", err)
			}

			if err := mm.RecordMigration(migrationName, batch, action); err != nil {
				tx.Rollback()
				return err
			}

			return tx.Commit().Error
		}
	} else {
		execFunc = func(sql string) error {
			if err := mm.gormDB.Exec(sql).Error; err != nil {
				return fmt.Errorf("migration execution failed: %w", err)
			}
			return mm.RecordMigration(migrationName, batch, action)
		}
	}

	if err := execFunc(sqlContent); err != nil {
		return err
	}

	return nil
}

// GetMigrationFiles returns every *.sql file under GetMigrationPath, sorted
// by filename (and therefore by timestamp prefix).
func (mm *MigrationManager) GetMigrationFiles() ([]string, error) {
	migrationPath := mm.GetMigrationPath()

	files, err := filepath.Glob(filepath.Join(migrationPath, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to get migration files: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

// Migrate runs every migration file not yet recorded as applied, in
// filename order, all under the same new batch number. If Options.Step is
// set, it stops once that many batches have been reached.
func (mm *MigrationManager) Migrate() error {
	if err := mm.InitMigrationTable(); err != nil {
		return err
	}

	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return err
	}

	nextBatch, err := mm.GetNextBatch()
	if err != nil {
		return err
	}

	migrationFiles, err := mm.GetMigrationFiles()
	if err != nil {
		return err
	}

	if len(migrationFiles) == 0 {
		return nil
	}

	currentMigrationMap := make(map[string]bool)
	for _, migration := range currentMigrations {
		currentMigrationMap[migration] = true
	}

	for _, filePath := range migrationFiles {
		filename := filepath.Base(filePath)
		migrationName := strings.TrimSuffix(filename, ".sql")

		if !currentMigrationMap[migrationName] {
			if err := mm.RunMigrationFile(filePath, "up", nextBatch); err != nil {
				return err
			}

			if mm.Options.Step > 0 && nextBatch >= mm.Options.Step {
				break
			}
		}
	}

	return nil
}

// Rollback undoes migrations in most-recently-applied-first order. steps
// <= 0 rolls back the whole last batch; steps > 0 rolls back that many
// individual migrations regardless of batch boundaries. A tracking row
// whose file is missing is just deleted rather than run.
func (mm *MigrationManager) Rollback(steps int) error {
	if err := mm.InitMigrationTable(); err != nil {
		return err
	}

	lastBatch, err := mm.GetLastBatch()
	if err != nil {
		return err
	}

	if lastBatch == 0 {
		return nil
	}

	var migrations []string
	var query string

	if steps <= 0 {
		query = fmt.Sprintf("SELECT migration FROM migrations WHERE batch = %d ORDER BY id DESC", lastBatch)
	} else {
		query = fmt.Sprintf("SELECT migration FROM migrations ORDER BY id DESC LIMIT %d", steps)
	}

	if err := mm.gormDB.Raw(query).Pluck("migration", &migrations).Error; err != nil {
		return fmt.Errorf("failed to get migrations to rollback: %w", err)
	}

	if len(migrations) == 0 {
		return nil
	}

	migrationPath := mm.GetMigrationPath()

	for _, migration := range migrations {
		filePath := filepath.Join(migrationPath, migration+".sql")

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := mm.gormDB.Where("migration = ?", migration).Delete(&Migrations{}).Error; err != nil {
				return fmt.Errorf("failed to remove orphaned migration record: %w", err)
			}
		} else {
			if err := mm.RunMigrationFile(filePath, "down", lastBatch); err != nil {
				return err
			}
		}
	}

	return nil
}

// Reset rolls back every applied migration.
func (mm *MigrationManager) Reset() error {
	return mm.Rollback(-1) // -1 indicates rollback all
}

// Refresh rolls back everything currently applied (Options.Step-limited,
// if set) and re-runs Migrate from scratch. It also cleans up orphaned
// tracking rows first so a deleted migration file doesn't block the
// rollback.
func (mm *MigrationManager) Refresh() error {
	if err := mm.InitMigrationTable(); err != nil {
		return err
	}

	if err := mm.CleanOrphanedMigrations(); err != nil {
		return err
	}

	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return err
	}

	if len(currentMigrations) == 0 {
		return mm.Migrate()
	}

	if err := mm.Rollback(mm.Options.Step); err != nil {
		return err
	}

	return mm.Migrate()
}

// Fresh drops and recreates the whole database (optionally dropping views/
// custom types first, per Options), then re-runs every migration from
// scratch. This is destructive - callers should confirm with the user
// before calling it.
func (mm *MigrationManager) Fresh() error {
	if mm.Options.DropViews {
		if err := mm.dropAllViews(); err != nil {
			return fmt.Errorf("failed to drop views: %w", err)
		}
	}

	if mm.dbType == PostgreSQL && mm.Options.DropTypes {
		if err := mm.dropCustomTypes(); err != nil {
			return fmt.Errorf("failed to drop custom types: %w", err)
		}
	}

	if err := mm.RecreateDatabase(); err != nil {
		return err
	}

	if err := mm.InitializeDatabase(mm.dbType); err != nil {
		return err
	}

	if err := mm.InitMigrationTable(); err != nil {
		return err
	}

	if err := mm.Migrate(); err != nil {
		return err
	}

	return nil
}

func (mm *MigrationManager) dropAllViews() error {
	switch mm.dbType {
	case MySQL:
		return mm.gormDB.Exec(`
            SELECT CONCAT('DROP VIEW IF EXISTS ', table_name, ';') 
            FROM information_schema.views 
            WHERE table_schema = DATABASE()
            INTO @sql;
            PREPARE stmt FROM @sql;
            EXECUTE stmt;
            DEALLOCATE PREPARE stmt;
        `).Error
	case PostgreSQL:
		return mm.gormDB.Exec(`
            DO $$ DECLARE
                r RECORD;
            BEGIN
                FOR r IN (SELECT schemaname || '.' || viewname AS view 
                          FROM pg_views 
                          WHERE schemaname NOT IN ('pg_catalog', 'information_schema')) 
                LOOP
                    EXECUTE 'DROP VIEW IF EXISTS ' || r.view || ' CASCADE';
                END LOOP;
            END $$;
        `).Error
	default:
		return fmt.Errorf("unsupported database type for view dropping")
	}
}

func (mm *MigrationManager) dropCustomTypes() error {
	if mm.dbType != PostgreSQL {
		return nil
	}

	return mm.gormDB.Exec(`
        DO $$ DECLARE
            r RECORD;
        BEGIN
            FOR r IN (SELECT n.nspname || '.' || t.typname AS type 
                      FROM pg_type t
                      JOIN pg_namespace n ON n.oid = t.typnamespace
                      WHERE t.typtype = 'c' 
                      AND n.nspname NOT IN ('pg_catalog', 'information_schema'))
            LOOP
                EXECUTE 'DROP TYPE IF EXISTS ' || r.type || ' CASCADE';
            END LOOP;
        END $$;
    `).Error
}

// Status returns each migration file's name mapped to "Ran" or "Pending".
func (mm *MigrationManager) Status() (map[string]string, error) {
	if err := mm.InitMigrationTable(); err != nil {
		return nil, err
	}

	migrationFiles, err := mm.GetMigrationFiles()
	if err != nil {
		return nil, err
	}

	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return nil, err
	}

	currentMigrationMap := make(map[string]bool)
	for _, migration := range currentMigrations {
		currentMigrationMap[migration] = true
	}

	statusMap := make(map[string]string)
	for _, filePath := range migrationFiles {
		filename := filepath.Base(filePath)
		migrationName := strings.TrimSuffix(filename, ".sql")

		if currentMigrationMap[migrationName] {
			statusMap[migrationName] = "Ran"
		} else {
			statusMap[migrationName] = "Pending"
		}
	}

	return statusMap, nil
}

// RecreateDatabase drops and recreates the target database. For
// PostgreSQL this has to happen over a separate connection to the
// "postgres" maintenance database, since you can't drop the database
// you're currently connected to - RecreateDatabase closes mm's own
// connection first and reconnects the caller is expected to redo via
// InitializeDatabase afterward (Fresh does this).
func (mm *MigrationManager) RecreateDatabase() error {
	var dbName string
	var dropQuery, createQuery string

	switch mm.dbType {
	case MySQL:
		dbName = mm.config.MySQL.DatabaseName
		charset := mm.config.MySQL.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		dropQuery = fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)
		createQuery = fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s", dbName, charset)
	case PostgreSQL:
		dbName = mm.config.PostgreSQL.DatabaseName

		if err := mm.sqlDB.Close(); err != nil {
			return fmt.Errorf("failed to close current connection: %w", err)
		}

		tempConfig := *mm.config
		tempConfig.PostgreSQL.DatabaseName = "postgres"
		tempGormDB, tempSqlDB, err := mm.dm.InitializeDatabase(PostgreSQL, &tempConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to postgres database: %w", err)
		}
		defer func() { _ = tempSqlDB.Close() }()

		terminateQuery := fmt.Sprintf(`
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, dbName)
		if err := tempGormDB.Exec(terminateQuery).Error; err != nil {
			return err
		}

		dropQuery = fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName)
		createQuery = fmt.Sprintf(`CREATE DATABASE "%s" WITH ENCODING = 'UTF8'`, dbName)

		if err := tempGormDB.Exec(dropQuery).Error; err != nil {
			return fmt.Errorf("failed to drop database: %w", err)
		}
		if err := tempGormDB.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("unsupported database type: %s", mm.dbType)
	}

	if err := mm.gormDB.Exec(dropQuery).Error; err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	if err := mm.gormDB.Exec(createQuery).Error; err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// MakeMigration writes a new timestamped migration file under
// GetMigrationPath, scaffolding a CREATE TABLE (Options.Create) or ALTER
// TABLE stub (Options.Table) if either is set, otherwise empty Up/Down
// sections.
func (mm *MigrationManager) MakeMigration(name string) error {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)
	filePath := filepath.Join(mm.GetMigrationPath(), filename)

	var contentBuilder strings.Builder
	contentBuilder.WriteString("-- +goose Up\n")
	contentBuilder.WriteString("-- SQL in this section is executed when the migration is applied\n\n")

	if mm.Options.Create != "" {
		contentBuilder.WriteString(mm.generateCreateTableSQL(mm.Options.Create))
	} else if mm.Options.Table != "" {
		contentBuilder.WriteString(mm.generateAlterTableSQL(mm.Options.Table))
	}

	contentBuilder.WriteString("\n-- +goose Down\n")
	contentBuilder.WriteString("-- SQL in this section is executed when the migration is rolled back\n\n")

	if mm.Options.Create != "" {
		fmt.Fprintf(&contentBuilder, "DROP TABLE IF EXISTS `%s`;\n", mm.Options.Create)
	} else if mm.Options.Table != "" {
		fmt.Fprintf(&contentBuilder, "-- Reverse your alter table operations for %s\n", mm.Options.Table)
	}

	if mm.Options.FullPath {
		fmt.Printf("Created migration: %s\n", filePath)
	}

	return os.WriteFile(filePath, []byte(contentBuilder.String()), 0644)
}

func (mm *MigrationManager) generateCreateTableSQL(tableName string) string {
	var sqlBuilder strings.Builder

	switch mm.dbType {
	case MySQL:
		fmt.Fprintf(&sqlBuilder, "CREATE TABLE IF NOT EXISTS `%s` (\n", tableName)
		sqlBuilder.WriteString("    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,\n")
		sqlBuilder.WriteString("    `created_at` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,\n")
		sqlBuilder.WriteString("    `updated_at` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP\n")
		sqlBuilder.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;\n")
	case PostgreSQL:
		fmt.Fprintf(&sqlBuilder, "CREATE TABLE IF NOT EXISTS \"%s\" (\n", tableName)
		sqlBuilder.WriteString("    \"id\" SERIAL PRIMARY KEY,\n")
		sqlBuilder.WriteString("    \"created_at\" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,\n")
		sqlBuilder.WriteString("    \"updated_at\" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP\n")
		sqlBuilder.WriteString(");\n")
	default:
		fmt.Fprintf(&sqlBuilder, "CREATE TABLE IF NOT EXISTS \"%s\" (\n", tableName)
		sqlBuilder.WriteString("    \"id\" SERIAL PRIMARY KEY,\n")
		sqlBuilder.WriteString("    \"created_at\" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n")
		sqlBuilder.WriteString("    \"updated_at\" TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n")
		sqlBuilder.WriteString(");\n")
	}

	return sqlBuilder.String()
}

func (mm *MigrationManager) generateAlterTableSQL(tableName string) string {
	return fmt.Sprintf("-- Add your ALTER TABLE statements for %s here\n", tableName)
}

// Close closes the manager's underlying *sql.DB, if any. Safe to call
// even if InitializeDatabase was never called.
func (mm *MigrationManager) Close() error {
	if mm.sqlDB != nil {
		return mm.sqlDB.Close()
	}
	return nil
}

// CountOrphanedMigrations counts tracking rows whose migration file no
// longer exists on disk, without removing them (see
// CleanOrphanedMigrations).
func (mm *MigrationManager) CountOrphanedMigrations() (int, error) {
	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return 0, fmt.Errorf("failed to get current migrations: %w", err)
	}

	migrationPath := mm.GetMigrationPath()
	orphanCount := 0

	for _, migration := range currentMigrations {
		filePath := filepath.Join(migrationPath, migration+".sql")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			orphanCount++
		}
	}

	return orphanCount, nil
}

// MigrateSpecific runs the single migration file whose name contains
// filePattern. Errors if zero or more than one file match. Unless force is
// true, it also errors if that migration was already applied; force skips
// that check and removes any existing tracking row before re-running it.
func (mm *MigrationManager) MigrateSpecific(filePattern string, force bool) error {
	if err := mm.InitMigrationTable(); err != nil {
		return err
	}

	nextBatch, err := mm.GetNextBatch()
	if err != nil {
		return err
	}

	migrationFiles, err := mm.GetMigrationFiles()
	if err != nil {
		return err
	}

	var matchedFiles []string
	for _, filePath := range migrationFiles {
		if strings.Contains(filepath.Base(filePath), filePattern) {
			matchedFiles = append(matchedFiles, filePath)
		}
	}

	switch len(matchedFiles) {
	case 0:
		return fmt.Errorf("no migration files matching '%s' found", filePattern)
	case 1:
		filePath := matchedFiles[0]
		filename := filepath.Base(filePath)
		migrationName := strings.TrimSuffix(filename, ".sql")

		if !force {
			var count int64
			if err := mm.gormDB.Model(&Migrations{}).
				Where("migration = ?", migrationName).
				Count(&count).Error; err != nil {
				return fmt.Errorf("failed to check migration status: %w", err)
			}

			if count > 0 {
				return fmt.Errorf("migration '%s' already executed", migrationName)
			}
		} else {
			if err := mm.gormDB.Where("migration = ?", migrationName).Delete(&Migrations{}).Error; err != nil {
				return fmt.Errorf("failed to remove existing migration record: %w", err)
			}
		}

		return mm.RunMigrationFile(filePath, "up", nextBatch)
	default:
		return fmt.Errorf("multiple files match '%s': %v", filePattern, matchedFiles)
	}
}

// GetDatabaseType returns the engine this manager is operating against.
func (mm *MigrationManager) GetDatabaseType() DatabaseType {
	return mm.dbType
}

// RemoveMigrationRecord deletes migrationName's tracking row without
// running its down SQL - used when the file itself is gone.
func (mm *MigrationManager) RemoveMigrationRecord(migrationName string) error {
	return mm.gormDB.Where("migration = ?", migrationName).Delete(&Migrations{}).Error
}
