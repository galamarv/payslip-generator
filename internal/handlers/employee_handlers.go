package handlers

import (
	"errors"
	"net/http"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SubmitAttendance(c *gin.Context) {
	var input struct {
		EmployeeID uint `json:"employeeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		c.JSON(http.StatusForbidden, gin.H{"error": "Attendance submission is not allowed on weekends."})
		return
	}

	var existingAttendance models.Attendance
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := database.DB.Where("employee_id = ? AND check_in >= ? AND check_in < ?", input.EmployeeID, startOfDay, endOfDay).First(&existingAttendance).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusConflict, gin.H{"error": "Attendance for today has already been submitted."})
		return
	}

	attendance := models.Attendance{
		EmployeeID: input.EmployeeID,
		CheckIn:    now,
		BaseModel: models.BaseModel{
			CreatedByID: input.EmployeeID,
			UpdatedByID: input.EmployeeID,
			RequestIP:   c.GetString("request_ip"),
		},
	}

	if err := database.DB.Create(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit attendance."})
		return
	}
	c.JSON(http.StatusCreated, attendance)
}

func SubmitOvertime(c *gin.Context) {
	var input struct {
		EmployeeID uint    `json:"employeeId" binding:"required"`
		Hours      float64 `json:"hours" binding:"required,gt=0,lte=3"`
		Date       string  `json:"date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if time.Now().Hour() < 17 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Overtime can only be proposed after 5 PM."})
		return
	}

	overtimeDate, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Please use YYYY-MM-DD."})
		return
	}

	overtime := models.Overtime{
		EmployeeID: input.EmployeeID,
		Hours:      input.Hours,
		Date:       overtimeDate,
		BaseModel: models.BaseModel{
			CreatedByID: input.EmployeeID,
			UpdatedByID: input.EmployeeID,
			RequestIP:   c.GetString("request_ip"),
		},
	}

	if err := database.DB.Create(&overtime).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit overtime."})
		return
	}
	c.JSON(http.StatusCreated, overtime)
}

func SubmitReimbursement(c *gin.Context) {
	var input struct {
		EmployeeID  uint    `json:"employeeId" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Description string  `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reimbursement := models.Reimbursement{
		EmployeeID:  input.EmployeeID,
		Amount:      input.Amount,
		Description: input.Description,
		BaseModel: models.BaseModel{
			CreatedByID: input.EmployeeID,
			UpdatedByID: input.EmployeeID,
			RequestIP:   c.GetString("request_ip"),
		},
	}

	if err := database.DB.Create(&reimbursement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit reimbursement."})
		return
	}
	c.JSON(http.StatusCreated, reimbursement)
}

func GeneratePayslip(c *gin.Context) {
	employeeIDStr := c.Query("employee_id")
	periodIDStr := c.Query("period_id")

	if employeeIDStr == "" || periodIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing employee_id or period_id query parameter"})
		return
	}

	employeeID, err1 := strconv.Atoi(employeeIDStr)
	periodID, err2 := strconv.Atoi(periodIDStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee_id or period_id"})
		return
	}

	var payslip models.Payslip
	err := database.DB.Where("employee_id = ? AND payroll_period_id = ?", employeeID, periodID).First(&payslip).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Payslip for this period not found."})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payslip."})
		return
	}

	c.JSON(http.StatusOK, payslip)
}
