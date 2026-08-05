package mysql

import (
	"database/sql"

	"gorm.io/gorm"
)

// UsersSeeder seeds the users table created by
// 20260805012446_create_users_table.sql.
type UsersSeeder struct{}

func (s *UsersSeeder) Name() string { return "UsersSeeder" }

func (s *UsersSeeder) Run(db *gorm.DB, sqlDB *sql.DB) error {
	rows := []map[string]any{
		{"name": "Rohman Beny Riyanto", "email": "beny@example.com", "role": "admin"},
		{"name": "Ada Lovelace", "email": "ada@example.com", "role": "member"},
		{"name": "Grace Hopper", "email": "grace@example.com", "role": "member"},
	}
	for _, row := range rows {
		if err := db.Table("users").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}
