package router_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"payslip-generator/internal/config"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"payslip-generator/internal/router"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testRouter *gin.Engine

func TestMain(m *testing.M) {
	// Setup
	gin.SetMode(gin.TestMode)
	config.LoadConfig() // Although we override DB, good practice to load others

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to set up test db for integration tests: %v", err)
	}
	database.DB = db
	// Add AuditLog to migrations for testing
	db.AutoMigrate(
		&models.Employee{}, &models.Admin{}, &models.Attendance{},
		&models.Overtime{}, &models.Reimbursement{}, &models.PayrollPeriod{},
		&models.Payslip{}, &models.AuditLog{},
	)

	testRouter = router.SetupRouter()

	// Run tests
	exitCode := m.Run()
	os.Exit(exitCode)
}

func performRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestFullPayrollFlow(t *testing.T) {
	// 1. Seed the database
	w_seed := performRequest(testRouter, "POST", "/seed", nil)
	if w_seed.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for seeding, got %d", w_seed.Code)
	}

	// 2. Admin creates a payroll period
	periodPayload := []byte(`{"startDate": "2025-06-01", "endDate": "2025-06-30", "adminId": 1}`)
	w_period := performRequest(testRouter, "POST", "/admin/payroll-periods", periodPayload)
	if w_period.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 for creating period, got %d", w_period.Code)
	}
	var periodResponse models.PayrollPeriod
	json.Unmarshal(w_period.Body.Bytes(), &periodResponse)
	if periodResponse.ID == 0 {
		t.Fatal("Failed to create payroll period, got zero ID")
	}

	// 3. Employee submits attendance
	attendancePayload := []byte(`{"employeeId": 5}`)
	w_att := performRequest(testRouter, "POST", "/employee/attendance", attendancePayload)
	if w_att.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 for submitting attendance, got %d", w_att.Code)
	}

	// 4. Admin runs payroll
	runPayload := []byte(`{"payrollPeriodId": 1, "adminId": 1}`)
	w_run := performRequest(testRouter, "POST", "/admin/run-payroll", runPayload)
	if w_run.Code != http.StatusAccepted {
		t.Fatalf("Expected status 202 for running payroll, got %d", w_run.Code)
	}
	// In a real app, you might need a small delay for the goroutine to finish
	time.Sleep(100 * time.Millisecond)

	// 5. Employee generates their payslip
	w_get_payslip := performRequest(testRouter, "GET", "/employee/payslip?employee_id=5&period_id=1", nil)
	if w_get_payslip.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for getting payslip, got %d. Body: %s", w_get_payslip.Code, w_get_payslip.Body.String())
	}
	var payslipResponse models.Payslip
	json.Unmarshal(w_get_payslip.Body.Bytes(), &payslipResponse)
	if payslipResponse.EmployeeID != 5 {
		t.Errorf("Expected payslip for employee 5, got for %d", payslipResponse.EmployeeID)
	}
	if payslipResponse.DaysAttended != 1 {
		t.Errorf("Expected 1 day of attendance to be recorded, got %d", payslipResponse.DaysAttended)
	}
	if payslipResponse.TakeHomePay <= 0 {
		t.Error("Expected positive take-home pay, but got zero or less")
	}

	// 6. Check that the audit log was created
	w_logs := performRequest(testRouter, "GET", "/admin/audit-logs", nil)
	if w_logs.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for getting audit logs, got %d", w_logs.Code)
	}
	var logsResponse []models.AuditLog
	json.Unmarshal(w_logs.Body.Bytes(), &logsResponse)
	if len(logsResponse) < 2 { // Should have at least one for CREATE_PERIOD and one for RAN_PAYROLL
		t.Fatalf("Expected at least two audit log entries, but got %d", len(logsResponse))
	}

	foundPayrollLog := false
	for _, log := range logsResponse {
		if log.Action == "RAN_PAYROLL" && log.UserID == 1 {
			foundPayrollLog = true
			break
		}
	}
	if !foundPayrollLog {
		t.Error("Expected to find an audit log for running payroll, but it was not found")
	}
}
