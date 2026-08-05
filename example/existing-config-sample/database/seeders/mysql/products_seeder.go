package mysql

import (
	"database/sql"

	"gorm.io/gorm"
)

// ProductsSeeder seeds the products table created by
// 20260805020000_create_products_table.sql.
type ProductsSeeder struct{}

func (s *ProductsSeeder) Name() string { return "ProductsSeeder" }

func (s *ProductsSeeder) Run(db *gorm.DB, sqlDB *sql.DB) error {
	rows := []map[string]any{
		{"name": "Mechanical Keyboard", "price_cents": 129900},
		{"name": "USB-C Hub", "price_cents": 34900},
	}
	for _, row := range rows {
		if err := db.Table("products").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}
