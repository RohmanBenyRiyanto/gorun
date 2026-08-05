package migration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RohmanBenyRiyanto/gorun/internal/engine"
)

// MigrationUtils holds the display/interaction helpers shared by
// StatusCommand and RunCommand - building the combined file+DB status
// view and driving the interactive follow-up menu.
type MigrationUtils struct{}

// NewMigrationUtils builds a MigrationUtils.
func NewMigrationUtils() *MigrationUtils {
	return &MigrationUtils{}
}

// GetMigrationDetails merges migration files on disk with tracking rows
// in the database into one MigrationStatus per migration - "Ran" (has
// both), "Pending" (file only), or "Missing" (tracking row only, file
// deleted).
func (mu *MigrationUtils) GetMigrationDetails(mm *engine.MigrationManager) ([]MigrationStatus, error) {
	var migrationStatuses []MigrationStatus

	migrationFiles, err := mm.GetMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get migration files: %w", err)
	}

	currentMigrations, err := mm.GetCurrentMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get current migrations: %w", err)
	}

	dbMigrations := make(map[string]engine.Migrations)
	for _, migration := range currentMigrations {
		var record engine.Migrations
		if err := mm.GetGormDB().Where("migration = ?", migration).First(&record).Error; err == nil {
			dbMigrations[migration] = record
		}
	}

	migrationMap := make(map[string]*MigrationStatus)
	counter := 1

	for _, filePath := range migrationFiles {
		filename := filepath.Base(filePath)
		migrationName := strings.TrimSuffix(filename, ".sql")

		status := &MigrationStatus{
			Number:      counter,
			Name:        migrationName,
			FileExists:  true,
			FilePath:    filePath,
			Description: mu.extractDescription(migrationName),
		}

		if fileInfo, err := os.Stat(filePath); err == nil {
			status.FileSize = formatFileSize(fileInfo.Size())
		}

		if content, err := mm.ParseMigrationFile(filePath); err == nil {
			status.HasUp = strings.TrimSpace(content.Up) != ""
			status.HasDown = strings.TrimSpace(content.Down) != ""
		}

		if dbRecord, exists := dbMigrations[migrationName]; exists {
			status.Status = "Ran"
			status.Batch = dbRecord.Batch
			status.RanAt = dbRecord.CreatedAt
		} else {
			status.Status = "Pending"
		}

		migrationMap[migrationName] = status
		counter++
	}

	for migrationName, dbRecord := range dbMigrations {
		if _, exists := migrationMap[migrationName]; !exists {
			status := &MigrationStatus{
				Number:      counter,
				Name:        migrationName,
				Status:      "Missing",
				Batch:       dbRecord.Batch,
				RanAt:       dbRecord.CreatedAt,
				FileExists:  false,
				Description: mu.extractDescription(migrationName),
			}
			migrationMap[migrationName] = status
			counter++
		}
	}

	for _, status := range migrationMap {
		migrationStatuses = append(migrationStatuses, *status)
	}

	sortMigrationsByName(migrationStatuses)

	for i := range migrationStatuses {
		migrationStatuses[i].Number = i + 1
	}

	return migrationStatuses, nil
}

// DisplayMigrationStatus prints a summary count, a detailed status table,
// and a legend for migrations.
func (mu *MigrationUtils) DisplayMigrationStatus(dbType engine.DatabaseType, dbName string, migrations []MigrationStatus) {
	engine.PrintSectionHeader(
		fmt.Sprintf("MIGRATION STATUS - %s DATABASE '%s'",
			strings.ToUpper(string(dbType)),
			dbName,
		),
	)
	fmt.Println()

	if len(migrations) == 0 {
		engine.PrintInfo("No migrations found")
		return
	}

	statusCounts := make(map[string]int)
	for _, migration := range migrations {
		statusCounts[migration.Status]++
	}

	engine.PrintDivider()
	engine.PrintTextH1("SUMMARY")
	fmt.Println()
	engine.PrintInfo("Total migrations: %d", len(migrations))
	if count, exists := statusCounts["Ran"]; exists {
		engine.PrintSuccess("Ran: %d", count)
	}
	if count, exists := statusCounts["Pending"]; exists {
		engine.PrintWarning("Pending: %d", count)
	}
	if count, exists := statusCounts["Missing"]; exists {
		engine.PrintError("Missing: %d", count)
	}
	fmt.Println()

	engine.PrintDivider()
	engine.PrintTextH1("DETAILED STATUS")
	fmt.Println()

	table := engine.NewTable([]string{"#", "Migration", "Status", "Batch", "Ran At", "Size", "Up", "Down"})

	for _, migration := range migrations {
		status := mu.getColoredStatus(migration.Status)

		batch := "-"
		if migration.Batch > 0 {
			batch = fmt.Sprintf("%d", migration.Batch)
		}

		ranAt := "-"
		if !migration.RanAt.IsZero() {
			ranAt = migration.RanAt.Format("2006-01-02 15:04")
		}

		size := "-"
		if migration.FileSize != "" {
			size = migration.FileSize
		}

		hasUp := engine.Green("✓")
		if !migration.HasUp {
			hasUp = engine.Red("✗")
		}

		hasDown := engine.Green("✓")
		if !migration.HasDown {
			hasDown = engine.Red("✗")
		}

		migrationName := migration.Name
		if len(migrationName) > 25 {
			migrationName = migrationName[:22] + "..."
		}

		table.AddRow([]string{
			fmt.Sprintf("%2d.", migration.Number),
			migrationName,
			status,
			batch,
			ranAt,
			size,
			hasUp,
			hasDown,
		})
	}

	table.SetColumnConfig(0, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 3, MaxWidth: 3})
	table.SetColumnConfig(1, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 25, MaxWidth: 30})
	table.SetColumnConfig(2, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 10})
	table.SetColumnConfig(3, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 6})
	table.SetColumnConfig(4, engine.ColumnConfig{HeaderAlign: engine.AlignLeft, ContentAlign: engine.AlignLeft, MinWidth: 16})
	table.SetColumnConfig(5, engine.ColumnConfig{HeaderAlign: engine.AlignRight, ContentAlign: engine.AlignRight, MinWidth: 8})
	table.SetColumnConfig(6, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 4})
	table.SetColumnConfig(7, engine.ColumnConfig{HeaderAlign: engine.AlignCenter, ContentAlign: engine.AlignCenter, MinWidth: 4})

	table.DrawHorizontal()
	fmt.Println()

	engine.PrintDivider()
	engine.PrintTextH1("LEGEND")
	legend := []string{
		"✓|Available / Yes",
		"✗|Not Available / No",
		"Ran|Migration has been executed",
		"Pending|Migration file exists but not executed",
		"Missing|Migration record exists but file is missing",
	}
	engine.PrintKeyValueTable(legend, engine.ColorForeground, engine.ColorReset)
}

// PromptMigrationActions shows the interactive post-status menu (run
// pending / clean orphaned / view a file / exit) and dispatches on the
// user's choice.
func (mu *MigrationUtils) PromptMigrationActions(mm *engine.MigrationManager, migrations []MigrationStatus) error {
	engine.PrintDivider()
	engine.PrintTextH1("ACTIONS")
	fmt.Println()
	engine.PrintOption(1, "Run pending migrations")
	engine.PrintOption(2, "Clean orphaned migrations")
	engine.PrintOption(3, "Show migration file content")
	engine.PrintOption(4, "Exit")
	fmt.Println()

	engine.PrintInputPrompt("Select action (1-4)", "4")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "1" && input != "2" && input != "3" && input != "4" {
		engine.PrintError("Input must be a number (1/2/3/4)")
		return nil
	}

	validInputs := []string{"1", "2", "3", "4"}
	isValid := false
	for _, valid := range validInputs {
		if input == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		engine.PrintError("Input must be a number (1/2/3/4)")
		return nil
	}

	switch input {
	case "1":
		return mu.runPendingMigrations(mm, migrations)
	case "2":
		return mu.cleanOrphanedMigrations(mm)
	case "3":
		return mu.showMigrationContent(migrations)
	case "4", "":
		engine.PrintInfo("Goodbye!")
		return nil
	default:
		engine.PrintError("Invalid selection")
		return mu.PromptMigrationActions(mm, migrations)
	}
}

func (mu *MigrationUtils) runPendingMigrations(mm *engine.MigrationManager, migrations []MigrationStatus) error {
	var pendingMigrations []string
	for _, migration := range migrations {
		if migration.Status == "Pending" {
			pendingMigrations = append(pendingMigrations, migration.Name)
		}
	}

	if len(pendingMigrations) == 0 {
		engine.PrintInfo("No pending migrations to run")
		return nil
	}

	engine.PrintWarning("Pending migrations to run (%d):", len(pendingMigrations))
	for _, name := range pendingMigrations {
		engine.PrintNormal("%s", "- "+name)
	}
	fmt.Println()

	engine.PrintInputPrompt("Run these migrations? (y/N)", "n")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		engine.PrintInfo("Running migrations...")
		if err := mm.Migrate(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		engine.PrintSuccess("All pending migrations executed successfully!")
	} else {
		engine.PrintInfo("Migration cancelled")
	}

	return nil
}

func (mu *MigrationUtils) cleanOrphanedMigrations(mm *engine.MigrationManager) error {
	engine.PrintInfo("Checking for orphaned migrations...")
	count, err := mm.CountOrphanedMigrations()
	if err != nil {
		return fmt.Errorf("failed to check orphaned migrations: %w", err)
	}

	if count == 0 {
		engine.PrintInfo("No orphaned migrations found")
		return nil
	}

	engine.PrintWarning("Found %d orphaned migrations", count)
	engine.PrintInputPrompt("Clean these migrations? (y/N)", "n")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		engine.PrintInfo("Clean operation cancelled")
		return nil
	}

	if input == "y" || input == "yes" {
		if err := mm.CleanOrphanedMigrations(); err != nil {
			return fmt.Errorf("failed to clean orphaned migrations: %w", err)
		}
		engine.PrintSuccess("Orphaned migrations cleaned successfully!")
	} else {
		engine.PrintInfo("Clean operation cancelled")
	}

	return nil
}

func (mu *MigrationUtils) showMigrationContent(migrations []MigrationStatus) error {
	engine.PrintDivider()
	engine.PrintTextH1("AVAILABLE MIGRATIONS")
	fmt.Println()

	var availableMigrations []MigrationStatus
	for _, m := range migrations {
		if m.FileExists {
			availableMigrations = append(availableMigrations, m)
			engine.PrintColoredNumFileDesc(m.Number, m.Name, m.Description)
		}
	}

	if len(availableMigrations) == 0 {
		engine.PrintInfo("No migration files available")
		return nil
	}

	fmt.Println()
	engine.PrintInputPrompt("Select migration number to view", "")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return fmt.Errorf("invalid selection")
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(availableMigrations) {
		return fmt.Errorf("invalid selection")
	}

	migration := availableMigrations[num-1]
	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	engine.PrintDivider()
	engine.PrintTextH1(fmt.Sprintf("MIGRATION: %s", migration.Name))
	engine.PrintInfo("File: %s", migration.FilePath)
	engine.PrintInfo("Size: %s", migration.FileSize)
	engine.PrintDivider()
	engine.PrintCodeBlock(string(content), migration.Name)
	engine.PrintDivider()

	return nil
}

func (mu *MigrationUtils) getColoredStatus(status string) string {
	switch status {
	case "Ran":
		return engine.Green(status)
	case "Pending":
		return engine.Yellow(status)
	case "Missing":
		return engine.Red(status)
	default:
		return status
	}
}

func (mu *MigrationUtils) extractDescription(migrationName string) string {
	parts := strings.SplitN(migrationName, "_", 2)
	if len(parts) > 1 {
		return strings.ReplaceAll(parts[1], "_", " ")
	}
	return "No description"
}

func sortMigrationsByName(migrations []MigrationStatus) {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}
