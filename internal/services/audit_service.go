package services

import (
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
)

// CreateAuditLog creates a new entry in the audit log table.
func CreateAuditLog(userID uint, userType, action, details, requestIP string) {
	logEntry := models.AuditLog{
		UserID:    userID,
		UserType:  userType,
		Action:    action,
		Details:   details,
		RequestIP: requestIP,
	}
	// This can be run in a goroutine so it doesn't block the main request flow.
	database.DB.Create(&logEntry)
}
