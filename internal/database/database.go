package database

import (
	"fmt"
	"log"
	"os"
	"payslip-generator/internal/models"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection pool.
var DB *gorm.DB

// SetupDatabase connects to PostgreSQL and runs auto-migrations.
func SetupDatabase() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection successful.")

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Employee{}, &models.Admin{}, &models.Attendance{},
		&models.Overtime{}, &models.Reimbursement{}, &models.PayrollPeriod{},
		&models.Payslip{}, &models.AuditLog{}, // Added AuditLog model
	)
	if err != nil {
		log.Fatal("Failed to migrate database schema:", err)
	}

	log.Println("Database migration successful.")
	DB = db
}
