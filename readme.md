# Payslip Generator API Documentation

This document provides a comprehensive overview of the Payslip Generator API, including its architecture, setup instructions, and a detailed guide to its endpoints.

## 1. Software Architecture

The application is built using Go and the Gin web framework, following a clean, modular architecture to ensure scalability and maintainability.

### Project Structure

The codebase is organized into a standard Go project layout:

```
/payslip-generator
├── cmd/server/main.go        # Application entry point. Initializes configs, DB, and router.
├── internal/
│   ├── config/               # Handles loading of environment variables.
│   ├── database/             # Manages the database connection (PostgreSQL) and schema migrations (GORM).
│   ├── handlers/             # Contains the Gin handlers that process HTTP requests.
│   ├── middleware/           # Custom middleware, such as the request logger for traceability.
│   ├── models/               # Defines the data structures (structs) for all database tables.
│   ├── router/               # Defines all API routes, groups them, and applies middleware.
│   └── services/             # Contains the core business logic (e.g., payroll calculation, audit logging).
├── go.mod                    # Defines the project module and dependencies.
└── .env                      # Stores configuration variables (not committed to Git).
```

### Key Design Choices

* **Go (Golang):** Chosen for its performance, simplicity, and strong concurrency features, making it ideal for a backend service that may handle complex calculations and many requests.
* **Gin Framework:** A minimalist, high-performance web framework for Go. It's used for routing and handling HTTP requests without unnecessary overhead, which is perfect for an API-centric service.
* **GORM:** The most popular ORM library for Go. It simplifies database interactions, allowing us to work with Go structs instead of raw SQL, which speeds up development and reduces errors. It also handles database migrations automatically.
* **PostgreSQL:** A powerful, open-source object-relational database system known for its reliability and data integrity, making it a safe choice for financial data.
* **Modular Design:** By separating concerns (database, routing, business logic), the application is easier to understand, test, and extend.
* **Audit Logging:** A dedicated `audit_logs` table and service (`internal/services/audit_service.go`) has been implemented to track significant events in the system, such as running payroll or creating payroll periods. This fulfills the "Plus Points" requirement for traceability.

### Example Data Flow (Submit Attendance)

1.  A `POST /employee/attendance` request hits the Gin router.
2.  The router passes the request to the `SubmitAttendance` function in the `handlers` package.
3.  The handler validates the request body and checks business rules (e.g., not a weekend).
4.  The handler interacts directly with the `database` package (using GORM) to query for existing records and create a new `Attendance` record.
5.  A success (or error) response is sent back to the client.

## 2. How-to Guide: Setup and Running

Follow these steps to set up and run the project on your local machine.

### Prerequisites

* **Go:** Version 1.21 or newer.
* **PostgreSQL:** A running instance of PostgreSQL.
* **Git:** For cloning the repository.

### Step-by-Step Setup

1.  **Clone the Repository (if applicable):**
    ```bash
    git clone <your-repository-url>
    cd payslip-generator
    ```

2.  **Create the `.env` File:**
    Create a file named `.env` in the project's root directory. Copy the contents from `.env.example` and fill in your PostgreSQL connection details.
    ```
    DB_HOST=localhost
    DB_USER=your_postgres_user
    DB_PASSWORD=your_postgres_password
    DB_NAME=payslip_db
    DB_PORT=5432
    ```

3.  **Create the Database:**
    Ensure you have created the database in PostgreSQL that you specified in your `.env` file (e.g., `payslip_db`).

4.  **Install Dependencies:**
    Open a terminal in the project root and run `go mod tidy`. This will download all the necessary libraries defined in `go.mod`.
    ```bash
    go mod tidy
    ```

5.  **Run the Application:**
    Navigate to the `cmd/server` directory and execute the `main.go` file.
    ```bash
    cd cmd/server
    go run main.go
    ```
    The server will start, connect to the database, run migrations, and listen for requests on `http://localhost:8080`.

## 3. API Usage

The following is a detailed guide for each API endpoint.

**Note on Authentication:** For simplicity, these endpoints pass `adminId` or `employeeId` in the request body. In a production environment, this is insecure. A proper implementation would involve a login endpoint that returns a JWT (JSON Web Token), which would then be included in the `Authorization` header of subsequent requests.

**Base URL:** `http://localhost:8080`

### 3.1. Seeding Endpoint

This endpoint populates the database with initial test data. **It should be run once after the initial setup.**

* **Endpoint:** `POST /seed`
* **Description:** Creates 1 admin user and 100 employee users with predefined credentials and salaries.
    * Admin Username: `admin`, Password: `admin`
    * Employee Usernames: `employee1`, `employee2`, ..., `employee100`
    * Employee Passwords: Same as username (e.g., `employee1`)
* **Request Body:** None
* **Example Request:**
    ```bash
    curl -X POST http://localhost:8080/seed
    ```
* **Success Response (200 OK):**
    ```json
    {
        "message": "Database seeded successfully with 1 admin and 100 employees."
    }
    ```

### 3.2. Admin Endpoints

These endpoints are for administrative tasks.

#### Create Payroll Period

* **Endpoint:** `POST /admin/payroll-periods`
* **Description:** Defines a new date range for a payroll run. Creates an audit log entry upon success.
* **Request Body:**
    ```json
    {
        "startDate": "YYYY-MM-DD",
        "endDate": "YYYY-MM-DD",
        "adminId": 1
    }
    ```
* **Example Request:**
    ```bash
    curl -X POST http://localhost:8080/admin/payroll-periods \
    -H "Content-Type: application/json" \
    -d '{"startDate": "2025-06-01", "endDate": "2025-06-30", "adminId": 1}'
    ```
* **Success Response (201 Created):**
    ```json
    {
        "id": 1,
        "createdAt": "2025-06-13T17:10:00.123Z",
        "updatedAt": "2025-06-13T17:10:00.123Z",
        "createdById": 1,
        "updatedById": 1,
        "startDate": "2025-06-01T00:00:00Z",
        "endDate": "2025-06-30T00:00:00Z",
        "isRun": false
    }
    ```

#### Run Payroll

* **Endpoint:** `POST /admin/run-payroll`
* **Description:** Initiates the payroll calculation for all employees for a given period. This is an asynchronous process. The server accepts the request and queues the calculation to run in the background, allowing the API to respond immediately. To check if the process is complete, you can either poll the 'Get Payslip Summary' endpoint or check the 'Get Audit Logs' endpoint for the 'RAN_PAYROLL' action. Creates an audit log entry upon completion.
* **Request Body:**
    ```json
    {
        "payrollPeriodId": 1,
        "adminId": 1
    }
    ```
* **Example Request:**
    ```bash
    curl -X POST http://localhost:8080/admin/run-payroll \
    -H "Content-Type: application/json" \
    -d '{"payrollPeriodId": 1, "adminId": 1}'
    ```
* **Success Response (202 Accepted):**
    ```json
    {
        "message": "Payroll run has been initiated. This may take a few moments."
    }
    ```

#### Get Payslip Summary

* **Endpoint:** `GET /admin/payslips/summary`
* **Description:** Retrieves a summary of all generated payslips for a specific period, including total payout.
* **Query Parameters:**
    * `period_id` (required): The ID of the payroll period.
* **Example Request:**
    ```bash
    curl -X GET "http://localhost:8080/admin/payslips/summary?period_id=1"
    ```
* **Success Response (200 OK):**
    ```json
    {
        "payrollPeriodId": 1,
        "totalPayout": 54559090.91,
        "employeePayslips": [
            {
                "employeeId": 1,
                "takeHomePay": 5000000
            },
            {
                "employeeId": 2,
                "takeHomePay": 5100000
            }
        ]
    }
    ```

#### Get Audit Logs

* **Endpoint:** `GET /admin/audit-logs`
* **Description:** Retrieves a list of all audit log entries, ordered by the most recent events first.
* **Query Parameters:** None
* **Example Request:**
    ```bash
    curl -X GET "http://localhost:8080/admin/audit-logs"
    ```
* **Success Response (200 OK):**
    ```json
    [
        {
            "id": 2,
            "createdAt": "2025-06-13T17:15:01.789Z",
            "userId": 1,
            "userType": "admin",
            "action": "RAN_PAYROLL",
            "details": "Successfully ran payroll for period ID 1.",
            "requestIp": "127.0.0.1"
        },
        {
            "id": 1,
            "createdAt": "2025-06-13T17:10:00.123Z",
            "userId": 1,
            "userType": "admin",
            "action": "CREATED_PERIOD",
            "details": "Created new payroll period ID 1 from 2025-06-01 to 2025-06-30.",
            "requestIp": "127.0.0.1"
        }
    ]
    ```

### 3.3. Employee Endpoints

These endpoints are for employees to manage their own data.

#### Submit Attendance

* **Endpoint:** `POST /employee/attendance`
* **Description:** Records a check-in for the employee for the current day. Cannot be submitted on weekends. Only one submission per day is allowed.
* **Request Body:**
    ```json
    {
        "employeeId": 10
    }
    ```
* **Example Request:**
    ```bash
    curl -X POST http://localhost:8080/employee/attendance \
    -H "Content-Type: application/json" \
    -d '{"employeeId": 10}'
    ```

#### Submit Overtime

* **Endpoint:** `POST /employee/overtime`
* **Description:** Submits a request for overtime hours. Limited to 3 hours per day and can only be submitted after 5 PM server time.
* **Request Body:**
    ```json
    {
        "employeeId": 10,
        "hours": 2,
        "date": "YYYY-MM-DD"
    }
    ```

#### Submit Reimbursement

* **Endpoint:** `POST /employee/reimbursements`
* **Description:** Submits a request for expense reimbursement.
* **Request Body:**
    ```json
    {
        "employeeId": 10,
        "amount": 75000,
        "description": "Taxi fare for client meeting"
    }
    ```

#### Generate Payslip

* **Endpoint:** `GET /employee/payslip`
* **Description:** Retrieves the detailed payslip for an employee for a specific period.
* **Query Parameters:**
    * `employee_id` (required): The employee's ID.
    * `period_id` (required): The payroll period's ID.
* **Example Request:**
    ```bash
    curl -X GET "http://localhost:8080/employee/payslip?employee_id=10&period_id=1"
    