package mysql

import (
	"database/sql"

	"gorm.io/gorm"
)

type NotesSeeder struct{}

func (s *NotesSeeder) Name() string { return "NotesSeeder" }

func (s *NotesSeeder) Run(db *gorm.DB, sqlDB *sql.DB) error {
	rows := []map[string]any{
		{"title": "First note", "body": "Single-engine, zero prompts, gorun picks MySQL automatically."},
		{"title": "Second note", "body": "See ../multi-db for what changes once a second engine is configured."},
	}
	for _, row := range rows {
		if err := db.Table("notes").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}
