package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel includes common fields for traceability.
type BaseModel struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedByID uint           `json:"createdById"`
	UpdatedByID uint           `json:"updatedById"`
	RequestIP   string         `json:"-"`
}

// Employee represents the employee data model.
type Employee struct {
	BaseModel
	Username string  `gorm:"unique;not null" json:"username"`
	Password string  `json:"-"`
	Salary   float64 `gorm:"not null" json:"salary"`
}

// Admin represents the admin user data model.
type Admin struct {
	BaseModel
	Username string `gorm:"unique;not null" json:"username"`
	Password string `json:"-"`
}

// Attendance represents an employee's daily attendance record.
type Attendance struct {
	BaseModel
	EmployeeID uint      `gorm:"not null;index" json:"employeeId"`
	CheckIn    time.Time `gorm:"not null" json:"checkIn"`
}

// Overtime represents an employee's overtime request.
type Overtime struct {
	BaseModel
	EmployeeID   uint      `gorm:"not null;index" json:"employeeId"`
	Date         time.Time `gorm:"type:date;not null" json:"date"`
	Hours        float64   `gorm:"not null" json:"hours"`
	IsApproved   bool      `gorm:"default:true" json:"isApproved"`
	PayrollRunID *uint     `gorm:"index" json:"payrollRunId,omitempty"`
}

// Reimbursement represents an employee's reimbursement request.
type Reimbursement struct {
	BaseModel
	EmployeeID   uint    `gorm:"not null;index" json:"employeeId"`
	Description  string  `gorm:"not null" json:"description"`
	Amount       float64 `gorm:"not null" json:"amount"`
	IsApproved   bool    `gorm:"default:true" json:"isApproved"`
	PayrollRunID *uint   `gorm:"index" json:"payrollRunId,omitempty"`
}

// PayrollPeriod defines the start and end dates for a payroll run.
type PayrollPeriod struct {
	BaseModel
	StartDate time.Time `gorm:"type:date;not null" json:"startDate"`
	EndDate   time.Time `gorm:"type:date;not null" json:"endDate"`
	IsRun     bool      `gorm:"default:false" json:"isRun"`
}

// Payslip stores the generated payslip details.
type Payslip struct {
	BaseModel
	EmployeeID      uint    `gorm:"not null;index" json:"employeeId"`
	PayrollPeriodID uint    `gorm:"not null;index" json:"payrollPeriodId"`
	BaseSalary      float64 `json:"baseSalary"`
	DaysAttended    int     `json:"daysAttended"`
	WorkingDays     int     `json:"workingDays"`
	ProratedSalary  float64 `json:"proratedSalary"`
	OvertimeHours   float64 `json:"overtimeHours"`
	OvertimePay     float64 `json:"overtimePay"`
	Reimbursement   float64 `json:"reimbursement"`
	TakeHomePay     float64 `json:"takeHomePay"`
	PayslipDetails  string  `gorm:"type:jsonb" json:"payslipDetails"`
}

// AuditLog tracks significant events in the system.
type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UserID    uint      `gorm:"index" json:"userId"`    // Admin or Employee ID
	UserType  string    `json:"userType"`               // "admin" or "employee"
	Action    string    `gorm:"not null" json:"action"` // e.g., "RAN_PAYROLL", "CREATED_PERIOD"
	Details   string    `json:"details"`                // e.g., "Ran payroll for period ID: 1"
	RequestIP string    `json:"requestIp"`
}
