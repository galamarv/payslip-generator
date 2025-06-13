package handlers

import (
	"fmt"
	"net/http"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"payslip-generator/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func CreatePayrollPeriod(c *gin.Context) {
	var input struct {
		StartDate string `json:"startDate" binding:"required"` // "YYYY-MM-DD"
		EndDate   string `json:"endDate" binding:"required"`
		AdminID   uint   `json:"adminId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err1 := time.Parse("2006-01-02", input.StartDate)
	endDate, err2 := time.Parse("2006-01-02", input.EndDate)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Please use YYYY-MM-DD."})
		return
	}

	period := models.PayrollPeriod{
		StartDate: startDate,
		EndDate:   endDate,
		BaseModel: models.BaseModel{
			CreatedByID: input.AdminID,
			UpdatedByID: input.AdminID,
			RequestIP:   c.GetString("request_ip"),
		},
	}

	if err := database.DB.Create(&period).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payroll period"})
		return
	}

	// Add an audit log entry
	details := fmt.Sprintf("Created new payroll period ID %d from %s to %s.", period.ID, input.StartDate, input.EndDate)
	go services.CreateAuditLog(input.AdminID, "admin", "CREATED_PERIOD", details, c.GetString("request_ip"))

	c.JSON(http.StatusCreated, period)
}

func RunPayroll(c *gin.Context) {
	var input struct {
		PayrollPeriodID uint `json:"payrollPeriodId" binding:"required"`
		AdminID         uint `json:"adminId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run the service in a goroutine for responsiveness
	go services.RunPayrollService(input.PayrollPeriodID, input.AdminID, c.GetString("request_ip"))

	c.JSON(http.StatusAccepted, gin.H{"message": "Payroll run has been initiated. This may take a few moments."})
}

func GetPayslipSummary(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	if periodIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing period_id query parameter"})
		return
	}
	periodID, err := strconv.Atoi(periodIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period_id"})
		return
	}

	var payslips []models.Payslip
	if err := database.DB.Where("payroll_period_id = ?", periodID).Find(&payslips).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch payslips"})
		return
	}

	if len(payslips) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No payslips found for this period. Has payroll been run?"})
		return
	}

	type EmployeeSummary struct {
		EmployeeID  uint    `json:"employeeId"`
		TakeHomePay float64 `json:"takeHomePay"`
	}

	var summaryList []EmployeeSummary
	totalPayout := 0.0

	for _, p := range payslips {
		summaryList = append(summaryList, EmployeeSummary{
			EmployeeID:  p.EmployeeID,
			TakeHomePay: p.TakeHomePay,
		})
		totalPayout += p.TakeHomePay
	}

	c.JSON(http.StatusOK, gin.H{
		"payrollPeriodId":  periodID,
		"totalPayout":      totalPayout,
		"employeePayslips": summaryList,
	})
}

// GetAuditLogs retrieves a list of all audit log entries.
func GetAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	// Order by most recent
	if err := database.DB.Order("created_at desc").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve audit logs"})
		return
	}
	c.JSON(http.StatusOK, logs)
}
