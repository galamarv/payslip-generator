package handlers

import (
	"fmt"
	"math"
	"net/http"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SeedDatabase creates the initial admin and employee records.
func SeedDatabase(c *gin.Context) {
	// Seed Admins - Password is "admin"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	admin := models.Admin{Username: "admin", Password: string(hashedPassword)}
	database.DB.FirstOrCreate(&admin, "username = ?", "admin")

	// Seed Employees - Password is the same as the username (e.g., "employee1")
	for i := 0; i < 100; i++ {
		username := fmt.Sprintf("employee%d", i+1)
		var employee models.Employee
		err := database.DB.FirstOrInit(&employee, models.Employee{Username: username}).Error
		if err == nil && employee.ID == 0 { // Only create if it doesn't exist
			employee.Salary = math.Round(5000000 + (float64(i) * 100000))

			// Set password to be the same as the username
			pass, _ := bcrypt.GenerateFromPassword([]byte(username), bcrypt.DefaultCost)
			employee.Password = string(pass)
			database.DB.Create(&employee)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Database seeded successfully with 1 admin and 100 employees."})
}
