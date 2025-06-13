package services

import (
	"log"
	"os"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testDB will hold the connection to our in-memory SQLite database.
var testDB *gorm.DB

// TestMain is a special function that runs before any tests in the package.
func TestMain(m *testing.M) {
	// Setup: Connect to the in-memory SQLite database.
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// We replace the global DB connection with our testDB connection.
	database.DB = testDB

	// Run migrations to create the schema in our test DB.
	err = testDB.AutoMigrate(
		&models.Employee{}, &models.Admin{}, &models.Attendance{},
		&models.Overtime{}, &models.Reimbursement{}, &models.PayrollPeriod{}, &models.Payslip{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate test database: %v", err)
	}

	log.Println("Test database setup complete.")

	// Run the tests.
	exitCode := m.Run()

	// Teardown: (Not strictly necessary for in-memory, but good practice).
	log.Println("Tearing down test database.")
	os.Exit(exitCode)
}

// cleanDB is a helper function to reset the database tables between tests.
func cleanDB() {
	testDB.Exec("DELETE FROM payslips")
	testDB.Exec("DELETE FROM reimbursements")
	testDB.Exec("DELETE FROM overtimes")
	testDB.Exec("DELETE FROM attendances")
	testDB.Exec("DELETE FROM payroll_periods")
	testDB.Exec("DELETE FROM employees")
}

func TestCalculatePayslipForEmployee(t *testing.T) {
	// Define a reusable employee and payroll period for our tests.
	employee := models.Employee{
		Username:  "testuser",
		Salary:    10500000, // 10.5M for easy calculation (500k/day for 21 working days)
		BaseModel: models.BaseModel{ID: 1},
	}
	// A period with 21 working days (e.g., June 2025)
	period := models.PayrollPeriod{
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		BaseModel: models.BaseModel{ID: 1},
	}

	t.Run("calculates correctly for full attendance", func(t *testing.T) {
		cleanDB()
		testDB.Create(&employee)
		testDB.Create(&period)
		for i := 1; i <= 30; i++ {
			day := time.Date(2025, 6, i, 9, 0, 0, 0, time.UTC)
			if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
				testDB.Create(&models.Attendance{EmployeeID: employee.ID, CheckIn: day})
			}
		}

		payslip, err := calculatePayslipForEmployee(employee, period, 1, "127.0.0.1")
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if payslip.TakeHomePay != 10500000 {
			t.Errorf("Expected takeHomePay of 10500000, but got %f", payslip.TakeHomePay)
		}
	})

	t.Run("calculates correctly with partial attendance, overtime, and reimbursement", func(t *testing.T) {
		cleanDB()
		testDB.Create(&employee)
		testDB.Create(&period)
		for i := 1; i <= 21; i++ { // Seed 15 days of attendance
			day := time.Date(2025, 6, i, 9, 0, 0, 0, time.UTC)
			if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
				testDB.Create(&models.Attendance{EmployeeID: employee.ID, CheckIn: day})
			}
		}
		testDB.Create(&models.Overtime{EmployeeID: employee.ID, Hours: 3, Date: time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)})
		testDB.Create(&models.Reimbursement{EmployeeID: employee.ID, Amount: 50000, Description: "Test"})

		payslip, err := calculatePayslipForEmployee(employee, period, 1, "127.0.0.1")

		expectedPay := 7925000.0
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if payslip.TakeHomePay != expectedPay {
			t.Errorf("Expected takeHomePay of %f, but got %f", expectedPay, payslip.TakeHomePay)
		}
	})
}
