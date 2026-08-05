package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

var (
	titleCaser = cases.Title(language.English)
	goModCache = struct {
		sync.Once
		moduleName string
		err        error
	}{}
)

// Seeder is one unit of seed data: Name identifies it for --class/--seeder
// targeting and GetSeederOrder, Run does the actual inserting against an
// open connection.
type Seeder interface {
	Name() string
	Run(db *gorm.DB, sqlDB *sql.DB) error
}

// SeederRegistry's GetSeederOrder is the authoritative run order - see
// SeederManager.GetSeederNames. Both MySQLRegistry and PostgreSQLRegistry
// already implement it.
type SeederRegistry interface {
	RegisterSeeder(typeName string) (reflect.Type, bool)
	GetSeederOrder() []string
}

// SeederManager discovers, orders, and runs the seeders for one database
// connection. Build one with NewSeederManager, passing the SeederRegistry
// your project implements (or nil if you only need CreateSeeder/scaffolding,
// not actually running anything).
type SeederManager struct {
	dbManager    *DatabaseManager
	dbType       DatabaseType
	seeders      map[string]Seeder
	config       *Config
	Options      SeederOptions
	registry     SeederRegistry
	isRegistered bool
	mu           sync.RWMutex
}

// SeederOptions configures a SeederManager run and where CreateSeeder
// writes new files.
type SeederOptions struct {
	Path     string
	RealPath bool
	FullPath bool
	Model    bool
	Table    string
	// Transaction wraps each seeder's Run in a DB transaction (default
	// true) - see RunSingleSeeder.
	Transaction bool
	// StopOnError halts on the first failing seeder when true (default,
	// matches historical behavior); when false, every seeder still runs
	// and failures are collected into one combined error at the end - see
	// RunSeeders.
	StopOnError bool
	// Only, if non-empty, restricts a run to just these seeder names
	// (order preserved from GetSeederOrder). Except removes names from
	// the run. Both apply to `seed run` with no --class/--seeder target.
	Only   []string
	Except []string
}

// NewSeederManager builds a SeederManager with Transaction and
// StopOnError both defaulted to true.
func NewSeederManager(dbManager *DatabaseManager, dbType DatabaseType, config *Config, registry SeederRegistry) *SeederManager {
	return &SeederManager{
		dbManager: dbManager,
		dbType:    dbType,
		config:    config,
		seeders:   make(map[string]Seeder),
		Options:   SeederOptions{Transaction: true, StopOnError: true},
		registry:  registry,
	}
}

// Register adds seeder to the in-memory set, keyed by its Name(). Usually
// called indirectly via RegisterAll/RegisterTargetSeeder rather than
// directly.
func (sm *SeederManager) Register(seeder Seeder) {
	sm.mu.Lock()
	sm.seeders[seeder.Name()] = seeder
	sm.mu.Unlock()
}

// GetSeeder returns the previously-registered seeder with this name, or
// nil if none was registered.
func (sm *SeederManager) GetSeeder(name string) Seeder {
	sm.mu.RLock()
	seeder := sm.seeders[name]
	sm.mu.RUnlock()
	return seeder
}

// GetSeederNames returns registered seeder names ordered by
// sm.registry.GetSeederOrder() - the authoritative dependency order (e.g.
// ParentRolesSeeder before SubRolesSeeder). Falls back to filename-sort
// order for anything GetSeederOrder doesn't mention, with a warning, so a
// forgotten registration never silently vanishes - it just runs last.
func (sm *SeederManager) GetSeederNames() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	fileNames := sm.fileSortedSeederNames()

	if sm.registry == nil {
		return fileNames
	}
	declared := sm.registry.GetSeederOrder()
	if len(declared) == 0 {
		return fileNames
	}

	fileSet := make(map[string]bool, len(fileNames))
	for _, n := range fileNames {
		fileSet[n] = true
	}

	seen := make(map[string]bool, len(declared))
	names := make([]string, 0, len(fileNames))
	for _, name := range declared {
		if fileSet[name] {
			names = append(names, name)
			seen[name] = true
			continue
		}
		PrintWarning("seeder %q is listed in GetSeederOrder but wasn't found among registered seeders (renamed or deleted?)", name)
	}
	for _, name := range fileNames {
		if seen[name] {
			continue
		}
		PrintWarning("seeder %q isn't listed in GetSeederOrder - running after declared seeders, in filename order. Add it to GetSeederOrder to control this.", name)
		names = append(names, name)
	}

	return names
}

// fileSortedSeederNames assumes sm.mu is already held (read) by the caller.
func (sm *SeederManager) fileSortedSeederNames() []string {
	seedersPath := sm.GetSeederPath()
	files, err := os.ReadDir(seedersPath)
	if err != nil {
		return nil
	}

	files = sortSeederFiles(files)

	names := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || file.Name() == "seeders.go" {
			continue
		}

		seederName := sm.GetSeederNameFromFilename(file.Name())
		if _, exists := sm.seeders[seederName]; exists {
			names = append(names, seederName)
		}
	}

	return names
}

// RunSingleSeeder runs one seeder with a progress spinner, wrapping it in
// a DB transaction unless Options.Transaction is false.
func (sm *SeederManager) RunSingleSeeder(seeder Seeder, db *gorm.DB, sqlDB *sql.DB) error {
	PrintDivider()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)

	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)

		loadingText := fmt.Sprintf("Running: %s", seeder.Name())
		FancyProgressBarContext(ctx, loadingText, "")
	}()

	go func() {
		PrintInfo("Starting: %s", seeder.Name())
		var err error
		if sm.Options.Transaction {
			err = db.Transaction(func(tx *gorm.DB) error {
				return seeder.Run(tx, sqlDB)
			})
		} else {
			err = seeder.Run(db, sqlDB)
		}
		done <- err
	}()

	err := <-done

	cancel()

	<-progressDone

	if err != nil {
		PrintWCmdError("seeder", "%s failed: %v", seeder.Name(), err)
		PrintDivider()
		return err
	}

	PrintSuccess("Completed: %s", seeder.Name())
	PrintDivider()
	return nil
}

// RunSeeders runs one named seeder (target != "") or every registered
// seeder in GetSeederNames order, filtered by Options.Only/Except. With
// Options.StopOnError false, a failing seeder doesn't abort the run - every
// seeder still executes and failures are joined into one returned error at
// the end.
func (sm *SeederManager) RunSeeders(db *gorm.DB, sqlDB *sql.DB, target string) error {
	if target != "" {
		return sm.runTargetSeeder(db, sqlDB, target)
	}

	PrintSectionHeader(fmt.Sprintf("RUNNING %s SEEDERS", strings.ToUpper(string(sm.dbType))))

	if err := sm.RegisterAll(); err != nil {
		return fmt.Errorf("failed to register seeders: %w", err)
	}

	seederNames := sm.filterSeederNames(sm.GetSeederNames())

	if len(seederNames) == 0 {
		PrintInfo("No seeders found to run")
		return nil
	}

	PrintInfo("Found %d seeder(s) to run", len(seederNames))
	PrintDivider()

	var failures []error
	for i, name := range seederNames {
		seeder := sm.GetSeeder(name)
		if seeder == nil {
			PrintWCmdError("seeder", "Seeder '%s' not found in registry", name)
			continue
		}

		PrintInfo("Running seeder %d of %d", i+1, len(seederNames))

		if err := sm.RunSingleSeeder(seeder, db, sqlDB); err != nil {
			wrapped := fmt.Errorf("seeder '%s' failed: %w", name, err)
			if sm.Options.StopOnError {
				return wrapped
			}
			failures = append(failures, wrapped)
		}

		if i < len(seederNames)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(failures) > 0 {
		PrintWCmdError("seeder", "%d of %d seeder(s) failed (continued past errors - --stop-on-error=false)", len(failures), len(seederNames))
		return errors.Join(failures...)
	}

	PrintSectionHeader("ALL SEEDERS COMPLETED SUCCESSFULLY")
	return nil
}

// filterSeederNames applies Options.Only/Except - Only restricts the run
// to just those names (order preserved from GetSeederOrder); Except
// removes them. If Only is set, Except is ignored (Only is the more
// specific ask).
func (sm *SeederManager) filterSeederNames(names []string) []string {
	if len(sm.Options.Only) > 0 {
		filtered := make([]string, 0, len(sm.Options.Only))
		for _, name := range names {
			if slices.Contains(sm.Options.Only, name) {
				filtered = append(filtered, name)
			}
		}
		return filtered
	}
	if len(sm.Options.Except) > 0 {
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			if !slices.Contains(sm.Options.Except, name) {
				filtered = append(filtered, name)
			}
		}
		return filtered
	}
	return names
}

func (sm *SeederManager) runTargetSeeder(db *gorm.DB, sqlDB *sql.DB, target string) error {
	// PrintInfo("Running target seeder: %s", target)

	if seeder := sm.GetSeeder(target); seeder != nil {
		return sm.RunSingleSeeder(seeder, db, sqlDB)
	}

	PrintInfo("Registering target seeder: %s", target)
	if err := sm.RegisterTargetSeeder(target); err != nil {
		PrintError("Failed to register target seeder: %v", err)
		return fmt.Errorf("failed to register target seeder: %w", err)
	}

	if seeder := sm.GetSeeder(target); seeder != nil {
		return sm.RunSingleSeeder(seeder, db, sqlDB)
	}

	return fmt.Errorf("seeder '%s' not found after registration", target)
}

// GetSeederPath resolves where seeder files live: Options.Path if set
// (cleaned unless RealPath), otherwise Config.MySQL.SeederPath or
// Config.PostgreSQL.SeederPath joined with "mysql"/"postgresql".
func (sm *SeederManager) GetSeederPath() string {
	if sm.Options.Path != "" {
		if sm.Options.RealPath {
			return sm.Options.Path
		}
		return filepath.Clean(sm.Options.Path)
	}

	switch sm.dbType {
	case MySQL:
		return filepath.Join(sm.config.MySQL.SeederPath, "mysql")
	case PostgreSQL:
		return filepath.Join(sm.config.PostgreSQL.SeederPath, "postgresql")
	default:
		return "./seeders"
	}
}

// CreateSeeder scaffolds a new seeder file under GetSeederPath, named
// after name with a timestamp prefix (matching migration file naming).
func (sm *SeederManager) CreateSeeder(name string) error {
	if name == "" {
		return fmt.Errorf("seeder name cannot be empty")
	}

	seederPath := sm.GetSeederPath()
	timestamp := time.Now().Format("20060102150405")
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	seederName := fmt.Sprintf("%sSeeder", toPascalCase(cleanName))
	filename := fmt.Sprintf("%s_%s_seeder.go", timestamp, cleanName)
	filePath := filepath.Join(seederPath, filename)

	PrintInfo("Creating new seeder in: %s", seederPath)
	FancyProgressBar("Setting up seeder structure", 500*time.Millisecond)

	if err := os.MkdirAll(seederPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", seederPath, err)
	}

	moduleName, err := getGoModuleName()
	if err != nil {
		return fmt.Errorf("failed to get Go module name: %w", err)
	}

	content := sm.generateSeederContent(moduleName, seederName, cleanName)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write seeder file: %w", err)
	}

	sm.printSeederCreationSummary(filename, seederName, cleanName, filePath)
	return nil
}

func (sm *SeederManager) printSeederCreationSummary(filename, seederName, cleanName, filePath string) {
	PrintSuccess("Successfully created seeder:")
	table := NewTable([]string{"Property", "Value"})
	table.AddRow([]string{"Filename", File(filename)})
	table.AddRow([]string{"Class", Function(seederName)})
	table.AddRow([]string{"Table", Keyword(cleanName)})
	table.DrawVertical()

	PrintTextNote("Next steps:")
	PrintDebug("  1. Implement your seeding logic in the Run() method")
	PrintDebug("  2. Create the corresponding model if needed")
	PrintDebug("  3. Register the seeder in the registry")

	if sm.Options.FullPath {
		if absPath, _ := filepath.Abs(filePath); absPath != "" {
			PrintInfo("Full path: %s", absPath)
		}
	}
}

func (sm *SeederManager) generateSeederContent(moduleName, seederName, tableName string) string {
	modelName := toPascalCase(tableName)
	driverImport := sm.dbManager.GetDriverImport(sm.dbType)
	packageName := string(sm.dbType)

	return fmt.Sprintf(`package %s

import (
	"database/sql"
	"reflect"

	"%s/internal/models"
	%s
	"gorm.io/gorm"
)

type %s struct{}

func (s *%s) Name() string {
	return reflect.Indirect(reflect.ValueOf(s)).Type().Name()
}

func (s *%s) Run(db *gorm.DB, sqlDB *sql.DB) error {
	data := s.builder()
	if len(data) == 0 {
		return nil
	}

	for _, item := range data {
		if err := db.Table(models.%s{}.TableName()).Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *%s) builder() []models.%s {
	return []models.%s{
		// Add your seed data here
	}
}
`, packageName, moduleName, driverImport, seederName, seederName, seederName, modelName, seederName, modelName, modelName)
}

// RegisterAll parses every seeder file under GetSeederPath and registers
// the ones whose struct name is known to the configured SeederRegistry.
// A no-op after the first successful call (guarded by isRegistered) - a
// file that fails to parse or match the registry is silently skipped
// rather than aborting the whole scan.
func (sm *SeederManager) RegisterAll() error {
	sm.mu.Lock()
	if sm.isRegistered {
		sm.mu.Unlock()
		return nil
	}
	sm.mu.Unlock()

	seedersPath := sm.GetSeederPath()
	files, err := os.ReadDir(seedersPath)
	if err != nil {
		return fmt.Errorf("failed to read seeders directory: %w", err)
	}

	files = sortSeederFiles(files)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || file.Name() == "seeders.go" {
			continue
		}

		if err := sm.registerSeederFromFile(filepath.Join(seedersPath, file.Name())); err != nil {
			continue
		}
	}

	sm.mu.Lock()
	sm.isRegistered = true
	sm.mu.Unlock()

	return nil
}

func (sm *SeederManager) registerSeederFromFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if _, isStruct := typeSpec.Type.(*ast.StructType); isStruct {
						structName := typeSpec.Name.Name
						if strings.HasSuffix(structName, "Seeder") {
							if typ, ok := sm.registry.RegisterSeeder(structName); ok {
								return sm.registerSeederByType(typ)
							}
						}
					}
				}
			}
		}
	}

	return fmt.Errorf("no valid seeder struct found")
}

func (sm *SeederManager) registerSeederByType(typ reflect.Type) error {
	seederValue := reflect.New(typ).Elem()
	seeder, ok := seederValue.Addr().Interface().(Seeder)
	if !ok {
		return fmt.Errorf("type %s does not implement Seeder interface", typ.Name())
	}
	sm.Register(seeder)
	return nil
}

// RegisterTargetSeeder registers just one seeder by name, without scanning
// the whole seeder directory first - tries the registry directly, then
// falls back to locating and parsing its file.
func (sm *SeederManager) RegisterTargetSeeder(target string) error {
	if typ, ok := sm.registry.RegisterSeeder(target); ok {
		return sm.registerSeederByType(typ)
	}

	filePath := sm.findSeederFileByTarget(target)
	if filePath == "" {
		return fmt.Errorf("seeder file not found for target: %s", target)
	}

	return sm.registerSeederFromFile(filePath)
}

func (sm *SeederManager) findSeederFileByTarget(target string) string {
	seedersPath := sm.GetSeederPath()
	files, _ := os.ReadDir(seedersPath)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || file.Name() == "seeders.go" {
			continue
		}

		if sm.matchesTarget(file.Name(), target) {
			return filepath.Join(seedersPath, file.Name())
		}
	}
	return ""
}

func (sm *SeederManager) matchesTarget(filename, target string) bool {
	if strings.EqualFold(strings.TrimSuffix(filename, ".go"), target) {
		return true
	}

	cleanName := strings.TrimSuffix(filename, ".go")
	if len(cleanName) > 15 && cleanName[14] == '_' {
		cleanName = cleanName[15:]
		if strings.EqualFold(cleanName, target) {
			return true
		}
	}

	return strings.EqualFold(sm.GetSeederNameFromFilename(filename), target)
}

// GetSeederNameFromFilename derives the expected seeder struct name from
// a seeder file's name, e.g. "20240101000000_parent_roles_seeder.go" ->
// "ParentRolesSeeder".
func (sm *SeederManager) GetSeederNameFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".go")
	if len(name) > 15 && name[14] == '_' {
		name = name[15:]
	}

	parts := strings.Split(name, "_")
	var pascalParts []string
	for _, part := range parts {
		if part != "seeder" {
			pascalParts = append(pascalParts, titleCaser.String(strings.ToLower(part)))
		}
	}
	return strings.Join(pascalParts, "") + "Seeder"
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = titleCaser.String(strings.ToLower(part))
		}
	}
	return strings.Join(parts, "")
}

func getGoModuleName() (string, error) {
	goModCache.Do(func() {
		currentDir, err := os.Getwd()
		if err != nil {
			goModCache.err = err
			return
		}

		for {
			goModPath := filepath.Join(currentDir, "go.mod")
			if _, err := os.Stat(goModPath); err == nil {
				data, err := os.ReadFile(goModPath)
				if err != nil {
					goModCache.err = err
					return
				}

				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "module ") {
						goModCache.moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
						return
					}
				}
			}

			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break
			}
			currentDir = parent
		}
		goModCache.err = fmt.Errorf("go.mod not found")
	})

	return goModCache.moduleName, goModCache.err
}

func sortSeederFiles(files []os.DirEntry) []os.DirEntry {
	sort.Slice(files, func(i, j int) bool {
		nameI := files[i].Name()
		nameJ := files[j].Name()

		if len(nameI) >= 14 && len(nameJ) >= 14 {
			timeI := nameI[:14]
			timeJ := nameJ[:14]

			if isAllDigits(timeI) && isAllDigits(timeJ) {
				return timeI < timeJ
			}
		}

		return nameI < nameJ
	})
	return files
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
