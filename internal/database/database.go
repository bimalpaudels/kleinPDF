package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Initialize initializes and returns a GORM database instance
func Initialize(databasePath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	return db, nil
}