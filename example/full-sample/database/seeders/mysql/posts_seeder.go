package mysql

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Post is a real GORM model for the posts table created by
// 20260805013000_create_posts_table.sql - unlike UsersSeeder's
// map[string]any rows, this shows seeding through a typed struct.
type Post struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Title     string
	Body      string
	CreatedAt time.Time
}

func (Post) TableName() string { return "posts" }

type PostsSeeder struct{}

func (s *PostsSeeder) Name() string { return "PostsSeeder" }

func (s *PostsSeeder) Run(db *gorm.DB, sqlDB *sql.DB) error {
	posts := []Post{
		{UserID: 1, Title: "Hello, gorun", Body: "First post seeded by PostsSeeder."},
		{UserID: 1, Title: "Migrations and seeders", Body: "Both tables came from real gorun commands."},
		{UserID: 2, Title: "Ada's post", Body: "Seeded referencing UsersSeeder's second row."},
	}
	return db.Create(&posts).Error
}
