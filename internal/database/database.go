package database

import (
	"fmt"
	"log"
	"time"

	"go-fiber/internal/config"
	"go-fiber/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.Customer{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Shop{},
		&models.Product{},
		&models.ProductVariant{},
		&models.ProductImage{},
		&models.Order{},
		&models.OrderItem{},
		&models.OrderStatusHistory{},
		&models.Payment{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
