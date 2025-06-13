package main

import (
	"log"
	"payslip-generator/internal/config"
	"payslip-generator/internal/database"
	"payslip-generator/internal/router"
)

func main() {
	// Load environment variables
	config.LoadConfig()

	// Initialize database
	database.SetupDatabase()

	// Setup and run the router
	r := router.SetupRouter()

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
