package services

import (
	"fmt"
	"log"
	"payslip-generator/internal/database"
	"payslip-generator/internal/models"
	"time"
)

// RunPayrollService orchestrates the entire payroll calculation process.
func RunPayrollService(periodID, adminID uint, requestIP string) {
	log.Printf("[Payroll Service] Starting payroll run for Period ID: %d by Admin ID: %d", periodID, adminID)

	var period models.PayrollPeriod
	if err := database.DB.First(&period, periodID).Error; err != nil {
		log.Printf("[Payroll Service] Error: Payroll Period %d not found.", periodID)
		return
	}

	if period.IsRun {
		log.Printf("[Payroll Service] Error: Payroll for Period %d has already been run.", periodID)
		return
	}

	database.DB.Model(&period).Update("is_run", true)

	var employees []models.Employee
	database.DB.Find(&employees)

	for _, emp := range employees {
		payslip, err := calculatePayslipForEmployee(emp, period, adminID, requestIP)
		if err != nil {
			log.Printf("[Payroll Service] Error calculating payslip for Employee ID %d: %v", emp.ID, err)
			continue
		}
		if err := database.DB.Create(&payslip).Error; err != nil {
			log.Printf("[Payroll Service] Error saving payslip for Employee ID %d: %v", emp.ID, err)
		} else {
			log.Printf("[Payroll Service] Successfully generated payslip for Employee ID %d.", emp.ID)
		}
	}

	log.Printf("[Payroll Service] Finished payroll run for Period ID: %d", periodID)

	// Add an audit log entry
	details := fmt.Sprintf("Successfully ran payroll for period ID %d.", periodID)
	CreateAuditLog(adminID, "admin", "RAN_PAYROLL", details, requestIP)
}

// calculatePayslipForEmployee contains the specific calculation logic for one employee.
func calculatePayslipForEmployee(emp models.Employee, period models.PayrollPeriod, adminID uint, requestIP string) (models.Payslip, error) {
	// 1. Calculate working days
	workingDays := 0
	currentDay := period.StartDate
	for !currentDay.After(period.EndDate) {
		if currentDay.Weekday() != time.Saturday && currentDay.Weekday() != time.Sunday {
			workingDays++
		}
		currentDay = currentDay.AddDate(0, 0, 1)
	}
	if workingDays == 0 {
		workingDays = 1 // Avoid division by zero
	}

	// 2. Count attendance
	var daysAttended int64
	database.DB.Model(&models.Attendance{}).
		Where("employee_id = ? AND check_in BETWEEN ? AND ?", emp.ID, period.StartDate, period.EndDate).
		Count(&daysAttended)

	// 3. Calculate Prorated Salary
	dailyRate := emp.Salary / float64(workingDays)
	proratedSalary := dailyRate * float64(daysAttended)

	// 4. Calculate Overtime
	var overtimes []models.Overtime
	database.DB.Where("employee_id = ? AND date BETWEEN ? AND ? AND payroll_run_id IS NULL", emp.ID, period.StartDate, period.EndDate).
		Find(&overtimes)
	totalOvertimeHours := 0.0
	for _, ot := range overtimes {
		totalOvertimeHours += ot.Hours
	}
	hourlyRate := dailyRate / 8
	overtimePay := totalOvertimeHours * (hourlyRate * 2)

	// 5. Calculate Reimbursements
	var reimbursements []models.Reimbursement
	database.DB.Where("employee_id = ? AND created_at BETWEEN ? AND ? AND payroll_run_id IS NULL", emp.ID, period.StartDate, period.EndDate).
		Find(&reimbursements)
	totalReimbursement := 0.0
	for _, r := range reimbursements {
		totalReimbursement += r.Amount
	}

	// 6. Calculate Take Home Pay
	takeHomePay := proratedSalary + overtimePay + totalReimbursement

	// 7. Assemble Details
	details := fmt.Sprintf(
		`{"attendance":{"daysAttended":%d,"totalWorkingDays":%d},"salary":{"base":%.2f,"prorated":%.2f},"overtime":{"hours":%.2f,"pay":%.2f},"reimbursements":{"total":%.2f}}`,
		daysAttended, workingDays, emp.Salary, proratedSalary, totalOvertimeHours, overtimePay, totalReimbursement,
	)

	payslip := models.Payslip{
		EmployeeID:      emp.ID,
		PayrollPeriodID: period.ID,
		BaseSalary:      emp.Salary,
		DaysAttended:    int(daysAttended),
		WorkingDays:     workingDays,
		ProratedSalary:  proratedSalary,
		OvertimeHours:   totalOvertimeHours,
		OvertimePay:     overtimePay,
		Reimbursement:   totalReimbursement,
		TakeHomePay:     takeHomePay,
		PayslipDetails:  details,
		BaseModel: models.BaseModel{
			CreatedByID: adminID,
			UpdatedByID: adminID,
			RequestIP:   requestIP,
		},
	}

	// Mark overtime and reimbursements as processed
	tx := database.DB.Begin()
	if len(overtimes) > 0 {
		var ids []uint
		for _, o := range overtimes {
			ids = append(ids, o.ID)
		}
		if err := tx.Model(&models.Overtime{}).Where("id IN ?", ids).Update("payroll_run_id", period.ID).Error; err != nil {
			tx.Rollback()
			return models.Payslip{}, err
		}
	}
	if len(reimbursements) > 0 {
		var ids []uint
		for _, r := range reimbursements {
			ids = append(ids, r.ID)
		}
		if err := tx.Model(&models.Reimbursement{}).Where("id IN ?", ids).Update("payroll_run_id", period.ID).Error; err != nil {
			tx.Rollback()
			return models.Payslip{}, err
		}
	}
	tx.Commit()

	return payslip, nil
}
