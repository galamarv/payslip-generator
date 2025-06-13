package router

import (
	"net/http"
	"payslip-generator/internal/handlers"
	"payslip-generator/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin router and defines all API endpoints.
func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())

	// A simple health check route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Payslip Generator API is running."})
	})

	// Public Endpoint to Seed Data
	r.POST("/seed", handlers.SeedDatabase)

	// Admin Routes
	admin := r.Group("/admin")
	{
		admin.POST("/payroll-periods", handlers.CreatePayrollPeriod)
		admin.POST("/run-payroll", handlers.RunPayroll)
		admin.GET("/payslips/summary", handlers.GetPayslipSummary)
		admin.GET("/audit-logs", handlers.GetAuditLogs) // New endpoint to view audit logs
	}

	// Employee Routes
	employee := r.Group("/employee")
	{
		employee.POST("/attendance", handlers.SubmitAttendance)
		employee.POST("/overtime", handlers.SubmitOvertime)
		employee.POST("/reimbursements", handlers.SubmitReimbursement)
		employee.GET("/payslip", handlers.GeneratePayslip)
	}

	return r
}
