package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestEnvironment configures a test router and in-memory DB.
func setupTestEnvironment() *gin.Engine {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	database.DB = db
	db.AutoMigrate(&models.Attendance{})

	r := gin.Default()
	return r
}

func TestSubmitAttendance(t *testing.T) {
	r := setupTestEnvironment()
	r.POST("/employee/attendance", SubmitAttendance)

	t.Run("should fail if attendance is submitted on a weekend", func(t *testing.T) {
		// This test is hard to make deterministic without mocking time.
		// For this example, we'll assume today is not a weekend.
		// In a real-world scenario, you would use a library to mock `time.Now()`.
		t.Skip("Skipping weekend test as it depends on the current day.")
	})

	t.Run("should fail if attendance is submitted twice on the same day", func(t *testing.T) {
		// Clean the table for a fresh start
		database.DB.Exec("DELETE FROM attendances")

		// First submission (should succeed)
		payload := []byte(`{"employeeId": 1}`)
		req, _ := http.NewRequest(http.MethodPost, "/employee/attendance", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected first submission to succeed with status 201, but got %d", w.Code)
		}

		// Second submission (should fail)
		req2, _ := http.NewRequest(http.MethodPost, "/employee/attendance", bytes.NewBuffer(payload))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Errorf("Expected second submission to fail with status 409, but got %d", w2.Code)
		}

		var response map[string]string
		json.Unmarshal(w2.Body.Bytes(), &response)
		expectedError := "Attendance for today has already been submitted."
		if response["error"] != expectedError {
			t.Errorf("Expected error message '%s', but got '%s'", expectedError, response["error"])
		}
	})
}
