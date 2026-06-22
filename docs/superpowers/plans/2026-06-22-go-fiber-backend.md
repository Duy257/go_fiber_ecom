# Go-Fiber Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng backend Go-Fiber với auth JWT, RBAC, Customer/User CRUD

**Architecture:** Monolithic single-binary, Clean Architecture (Handler → Service → Repository). GORM cho PostgreSQL, JWT stateless cho auth.

**Tech Stack:** Go, Fiber v2, GORM, PostgreSQL, golang-jwt/v5, bcrypt

---

## File Structure

```
go-fiber/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── database/database.go
│   ├── middleware/auth.go
│   ├── middleware/rbac.go
│   ├── models/customer.go
│   ├── models/user.go
│   ├── models/role.go
│   ├── models/permission.go
│   ├── repositories/customer_repo.go
│   ├── repositories/user_repo.go
│   ├── repositories/role_repo.go
│   ├── repositories/permission_repo.go
│   ├── services/auth_service.go
│   ├── services/customer_service.go
│   ├── services/user_service.go
│   ├── services/role_service.go
│   ├── services/dashboard_service.go
│   ├── handlers/auth_handler.go
│   ├── handlers/customer_handler.go
│   ├── handlers/user_handler.go
│   ├── handlers/role_handler.go
│   ├── handlers/permission_handler.go
│   ├── handlers/dashboard_handler.go
│   └── utils/
│       ├── response.go
│       ├── password.go
│       └── validator.go
├── .env.example
└── go.mod
```

---

### Task 1: Project Setup & Dependencies

**Files:**
- Create: `go.mod`
- Create: `.env.example`
- Create: `cmd/server/main.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/duynguyen/MyProject/golang/go-fiber
go mod init go-fiber
```

- [ ] **Step 2: Create .env.example**

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=go_fiber
JWT_SECRET=your-secret-key-change-in-production
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
SERVER_PORT=3000
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=admin123
ADMIN_PHONE=0900000000
```

- [ ] **Step 3: Install dependencies**

```bash
go get github.com/gofiber/fiber/v2
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
go get github.com/google/uuid
go get github.com/joho/godotenv
go get github.com/go-playground/validator/v10
```

- [ ] **Step 4: Create minimal main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("Go Fiber API starting...")
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./cmd/server/
```

- [ ] **Step 6: Commit**

```bash
git init
git add .
git commit -m "chore: initialize project with dependencies"
```

---

### Task 2: Config

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: Create config struct and loader**

```go
package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	ServerPort    string
	AdminEmail    string
	AdminPassword string
	AdminPhone    string
}

func Load() *Config {
	godotenv.Load()

	accessTTL, _ := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	refreshTTL, _ := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))

	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "go_fiber"),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		JWTAccessTTL:  accessTTL,
		JWTRefreshTTL: refreshTTL,
		ServerPort:    getEnv("SERVER_PORT", "3000"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		AdminPhone:    getEnv("ADMIN_PHONE", "0900000000"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add config loader"
```

---

### Task 3: Utils (Response, Password, Validator)

**Files:**
- Create: `internal/utils/response.go`
- Create: `internal/utils/password.go`
- Create: `internal/utils/validator.go`

- [ ] **Step 1: Create response.go**

```go
package utils

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Message    string      `json:"message,omitempty"`
	Error      *ErrorData  `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func Success(c *fiber.Ctx, data interface{}, message string) error {
	return c.JSON(Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func SuccessWithPagination(c *fiber.Ctx, data interface{}, page, limit int, total int64) error {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return c.JSON(Response{
		Success: true,
		Data:    data,
		Pagination: &Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func Error(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(Response{
		Success: false,
		Error: &ErrorData{
			Code:    code,
			Message: message,
		},
	})
}
```

- [ ] **Step 2: Create password.go**

```go
package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
```

- [ ] **Step 3: Create validator.go**

```go
package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field())
		switch e.Tag() {
		case "required":
			errors[field] = fmt.Sprintf("%s is required", field)
		case "email":
			errors[field] = fmt.Sprintf("%s must be a valid email", field)
		case "min":
			errors[field] = fmt.Sprintf("%s must be at least %s characters", field, e.Param())
		default:
			errors[field] = fmt.Sprintf("%s is invalid", field)
		}
	}
	return errors
}

func IsValidEmailOrPhone(login string) bool {
	if strings.Contains(login, "@") {
		return validate.Var(login, "email") == nil
	}
	return len(login) >= 9 && len(login) <= 15
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/utils/
git commit -m "feat: add utils (response, password, validator)"
```

---

### Task 4: Models

**Files:**
- Create: `internal/models/customer.go`
- Create: `internal/models/user.go`
- Create: `internal/models/role.go`
- Create: `internal/models/permission.go`

- [ ] **Step 1: Create role.go**

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string       `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
```

- [ ] **Step 2: Create permission.go**

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
```

- [ ] **Step 3: Create customer.go**

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email       *string   `gorm:"type:varchar(255);uniqueIndex" json:"email,omitempty"`
	PhoneNumber *string   `gorm:"type:varchar(20);uniqueIndex" json:"phone_number,omitempty"`
	Password    string    `gorm:"type:varchar(255);not null" json:"-"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	Status      string    `gorm:"type:varchar(20);default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

- [ ] **Step 4: Create user.go**

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email       *string   `gorm:"type:varchar(255);uniqueIndex" json:"email,omitempty"`
	PhoneNumber *string   `gorm:"type:varchar(20);uniqueIndex" json:"phone_number,omitempty"`
	Password    string    `gorm:"type:varchar(255);not null" json:"-"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	RoleID      uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
	Role        Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Status      string    `gorm:"type:varchar(20);default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/models/
git commit -m "feat: add models (customer, user, role, permission)"
```

---

### Task 5: Database Connection & Migration

**Files:**
- Create: `internal/database/database.go`

- [ ] **Step 1: Create database.go**

```go
package database

import (
	"fmt"
	"log"

	"go-fiber/internal/config"
	"go-fiber/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	return db
}

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.Customer{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/database/
git commit -m "feat: add database connection and migration"
```

---

### Task 6: Repositories

**Files:**
- Create: `internal/repositories/customer_repo.go`
- Create: `internal/repositories/user_repo.go`
- Create: `internal/repositories/role_repo.go`
- Create: `internal/repositories/permission_repo.go`

- [ ] **Step 1: Create role_repo.go**

```go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) FindAll() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}

func (r *RoleRepository) FindByID(id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, "id = ?", id).Error
	return &role, err
}

func (r *RoleRepository) FindByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, "name = ?", name).Error
	return &role, err
}

func (r *RoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

func (r *RoleRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Role{}, "id = ?", id).Error
}
```

- [ ] **Step 2: Create permission_repo.go**

```go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) FindAll() ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Find(&permissions).Error
	return permissions, err
}

func (r *PermissionRepository) FindByIDs(ids []uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Where("id IN ?", ids).Find(&permissions).Error
	return permissions, err
}

func (r *PermissionRepository) Create(permission *models.Permission) error {
	return r.db.Create(permission).Error
}

func (r *PermissionRepository) FindByName(name string) (*models.Permission, error) {
	var perm models.Permission
	err := r.db.First(&perm, "name = ?", name).Error
	return &perm, err
}
```

- [ ] **Step 3: Create customer_repo.go**

```go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) FindByEmailOrPhone(login string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("email = ? OR phone_number = ?", login, login).First(&customer).Error
	return &customer, err
}

func (r *CustomerRepository) FindByID(id uuid.UUID) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.First(&customer, "id = ?", id).Error
	return &customer, err
}

func (r *CustomerRepository) FindAll(page, limit int) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	r.db.Model(&models.Customer{}).Count(&total)
	err := r.db.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&customers).Error
	return customers, total, err
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	return r.db.Create(customer).Error
}

func (r *CustomerRepository) Update(customer *models.Customer) error {
	return r.db.Save(customer).Error
}

func (r *CustomerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Customer{}, "id = ?", id).Error
}
```

- [ ] **Step 4: Create user_repo.go**

```go
package repositories

import (
	"go-fiber/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmailOrPhone(login string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Role.Permissions").Where("email = ? OR phone_number = ?", login, login).First(&user).Error
	return &user, err
}

func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Role.Permissions").First(&user, "id = ?", id).Error
	return &user, err
}

func (r *UserRepository) FindAll(page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	r.db.Model(&models.User{}).Count(&total)
	err := r.db.Preload("Role").Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/repositories/
git commit -m "feat: add repositories (customer, user, role, permission)"
```

---

### Task 7: Services

**Files:**
- Create: `internal/services/auth_service.go`
- Create: `internal/services/customer_service.go`
- Create: `internal/services/user_service.go`
- Create: `internal/services/role_service.go`
- Create: `internal/services/dashboard_service.go`

- [ ] **Step 1: Create auth_service.go**

```go
package services

import (
	"errors"
	"time"

	"go-fiber/internal/config"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService struct {
	cfg         *config.Config
	userRepo    *repositories.UserRepository
	customerRepo *repositories.CustomerRepository
}

func NewAuthService(cfg *config.Config, userRepo *repositories.UserRepository, customerRepo *repositories.CustomerRepository) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, customerRepo: customerRepo}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *AuthService) LoginAdmin(login, password string) (*TokenPair, error) {
	user, err := s.userRepo.FindByEmailOrPhone(login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	return s.generateTokenPair(user.ID.String(), "admin")
}

func (s *AuthService) LoginCustomer(login, password string) (*TokenPair, error) {
	customer, err := s.customerRepo.FindByEmailOrPhone(login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, customer.Password) {
		return nil, errors.New("invalid credentials")
	}

	if customer.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	return s.generateTokenPair(customer.ID.String(), "customer")
}

func (s *AuthService) Refresh(refreshToken string) (string, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		return "", errors.New("invalid token type")
	}

	sub, _ := claims["sub"].(string)
	roleType, _ := claims["role"].(string)

	return s.generateAccessToken(sub, roleType)
}

func (s *AuthService) generateTokenPair(sub, roleType string) (*TokenPair, error) {
	accessToken, err := s.generateAccessToken(sub, roleType)
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"type": "refresh",
		"role": roleType,
		"exp":  time.Now().Add(s.cfg.JWTRefreshTTL).Unix(),
	})

	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshStr,
	}, nil
}

func (s *AuthService) generateAccessToken(sub, roleType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"type": "access",
		"role": roleType,
		"exp":  time.Now().Add(s.cfg.JWTAccessTTL).Unix(),
	})

	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) GetUserFromToken(userID, roleType string) (interface{}, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	if roleType == "admin" {
		return s.userRepo.FindByID(uid)
	}
	return s.customerRepo.FindByID(uid)
}
```

- [ ] **Step 2: Create customer_service.go**

```go
package services

import (
	"errors"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerService struct {
	repo *repositories.CustomerRepository
}

func NewCustomerService(repo *repositories.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

type CreateCustomerInput struct {
	Email       string `json:"email" validate:"omitempty,email"`
	PhoneNumber string `json:"phone_number" validate:"omitempty"`
	Password    string `json:"password" validate:"required,min=6"`
	Name        string `json:"name" validate:"required"`
}

type UpdateCustomerInput struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Name        *string `json:"name"`
	Status      *string `json:"status"`
}

func (s *CustomerService) GetAll(page, limit int) ([]models.Customer, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *CustomerService) GetByID(id uuid.UUID) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *CustomerService) Create(input CreateCustomerInput) (*models.Customer, error) {
	if input.Email == "" && input.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	customer := &models.Customer{
		Password: hashedPassword,
		Name:     input.Name,
		Status:   "active",
	}

	if input.Email != "" {
		customer.Email = &input.Email
	}
	if input.PhoneNumber != "" {
		customer.PhoneNumber = &input.PhoneNumber
	}

	if err := s.repo.Create(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) Update(id uuid.UUID, input UpdateCustomerInput) (*models.Customer, error) {
	customer, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	if input.Email != nil {
		customer.Email = input.Email
	}
	if input.PhoneNumber != nil {
		customer.PhoneNumber = input.PhoneNumber
	}
	if input.Name != nil {
		customer.Name = *input.Name
	}
	if input.Status != nil {
		customer.Status = *input.Status
	}

	if err := s.repo.Update(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("customer not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 3: Create user_service.go**

```go
package services

import (
	"errors"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	repo     *repositories.UserRepository
	roleRepo *repositories.RoleRepository
}

func NewUserService(repo *repositories.UserRepository, roleRepo *repositories.RoleRepository) *UserService {
	return &UserService{repo: repo, roleRepo: roleRepo}
}

type CreateUserInput struct {
	Email       string `json:"email" validate:"omitempty,email"`
	PhoneNumber string `json:"phone_number" validate:"omitempty"`
	Password    string `json:"password" validate:"required,min=6"`
	Name        string `json:"name" validate:"required"`
	RoleID      string `json:"role_id" validate:"required"`
}

type UpdateUserInput struct {
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Name        *string `json:"name"`
	RoleID      *string `json:"role_id"`
	Status      *string `json:"status"`
}

func (s *UserService) GetAll(page, limit int) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.FindAll(page, limit)
}

func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *UserService) Create(input CreateUserInput) (*models.User, error) {
	if input.Email == "" && input.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}

	roleID, err := uuid.Parse(input.RoleID)
	if err != nil {
		return nil, errors.New("invalid role_id")
	}

	_, err = s.roleRepo.FindByID(roleID)
	if err != nil {
		return nil, errors.New("role not found")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Password: hashedPassword,
		Name:     input.Name,
		RoleID:   roleID,
		Status:   "active",
	}

	if input.Email != "" {
		user.Email = &input.Email
	}
	if input.PhoneNumber != "" {
		user.PhoneNumber = &input.PhoneNumber
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(id uuid.UUID, input UpdateUserInput) (*models.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if input.Email != nil {
		user.Email = input.Email
	}
	if input.PhoneNumber != nil {
		user.PhoneNumber = input.PhoneNumber
	}
	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Status != nil {
		user.Status = *input.Status
	}
	if input.RoleID != nil {
		roleID, err := uuid.Parse(*input.RoleID)
		if err != nil {
			return nil, errors.New("invalid role_id")
		}
		user.RoleID = roleID
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 4: Create role_service.go**

```go
package services

import (
	"errors"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleService struct {
	repo           *repositories.RoleRepository
	permissionRepo *repositories.PermissionRepository
}

func NewRoleService(repo *repositories.RoleRepository, permissionRepo *repositories.PermissionRepository) *RoleService {
	return &RoleService{repo: repo, permissionRepo: permissionRepo}
}

type CreateRoleInput struct {
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

type UpdateRoleInput struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

func (s *RoleService) GetAll() ([]models.Role, error) {
	return s.repo.FindAll()
}

func (s *RoleService) GetByID(id uuid.UUID) (*models.Role, error) {
	return s.repo.FindByID(id)
}

func (s *RoleService) Create(input CreateRoleInput) (*models.Role, error) {
	role := &models.Role{
		Name:        input.Name,
		Description: input.Description,
	}

	if len(input.PermissionIDs) > 0 {
		permIDs := make([]uuid.UUID, len(input.PermissionIDs))
		for i, pid := range input.PermissionIDs {
			id, err := uuid.Parse(pid)
			if err != nil {
				return nil, errors.New("invalid permission_id: " + pid)
			}
			permIDs[i] = id
		}

		permissions, err := s.permissionRepo.FindByIDs(permIDs)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
	}

	if err := s.repo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) Update(id uuid.UUID, input UpdateRoleInput) (*models.Role, error) {
	role, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, err
	}

	if input.Name != nil {
		role.Name = *input.Name
	}
	if input.Description != nil {
		role.Description = *input.Description
	}

	if input.PermissionIDs != nil {
		permIDs := make([]uuid.UUID, len(input.PermissionIDs))
		for i, pid := range input.PermissionIDs {
			id, err := uuid.Parse(pid)
			if err != nil {
				return nil, errors.New("invalid permission_id: " + pid)
			}
			permIDs[i] = id
		}

		permissions, err := s.permissionRepo.FindByIDs(permIDs)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
	}

	if err := s.repo.Update(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("role not found")
		}
		return err
	}
	return s.repo.Delete(id)
}
```

- [ ] **Step 5: Create dashboard_service.go**

```go
package services

import (
	"go-fiber/internal/models"

	"gorm.io/gorm"
)

type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

type DashboardStats struct {
	TotalCustomers int64 `json:"total_customers"`
	TotalUsers     int64 `json:"total_users"`
	TotalRoles     int64 `json:"total_roles"`
	ActiveCustomers int64 `json:"active_customers"`
}

func (s *DashboardService) GetStats() (*DashboardStats, error) {
	var stats DashboardStats

	s.db.Model(&models.Customer{}).Count(&stats.TotalCustomers)
	s.db.Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.Model(&models.Role{}).Count(&stats.TotalRoles)
	s.db.Model(&models.Customer{}).Where("status = ?", "active").Count(&stats.ActiveCustomers)

	return &stats, nil
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/services/
git commit -m "feat: add services (auth, customer, user, role, dashboard)"
```

---

### Task 8: Middleware

**Files:**
- Create: `internal/middleware/auth.go`
- Create: `internal/middleware/rbac.go`

- [ ] **Step 1: Create auth.go**

```go
package middleware

import (
	"strings"

	"go-fiber/internal/config"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid authorization format")
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid or expired token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid token claims")
		}

		tokenType, _ := claims["type"].(string)
		if tokenType != "access" {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid token type")
		}

		sub, _ := claims["sub"].(string)
		role, _ := claims["role"].(string)

		c.Locals("userID", sub)
		c.Locals("userRole", role)

		return c.Next()
	}
}
```

- [ ] **Step 2: Create rbac.go**

```go
package middleware

import (
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func RequirePermission(userRepo *repositories.UserRepository, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok || userRole != "admin" {
			return utils.Error(c, 403, "FORBIDDEN", "Admin access required")
		}

		userID, ok := c.Locals("userID").(string)
		if !ok {
			return utils.Error(c, 401, "UNAUTHORIZED", "User not found in context")
		}

		uid, err := uuid.Parse(userID)
		if err != nil {
			return utils.Error(c, 401, "UNAUTHORIZED", "Invalid user ID")
		}

		user, err := userRepo.FindByID(uid)
		if err != nil {
			return utils.Error(c, 401, "UNAUTHORIZED", "User not found")
		}

		if user.Role.Name == "super_admin" {
			return c.Next()
		}

		for _, perm := range user.Role.Permissions {
			if perm.Name == permission {
				return c.Next()
			}
		}

		return utils.Error(c, 403, "FORBIDDEN", "Permission denied: "+permission)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/
git commit -m "feat: add JWT auth and RBAC middleware"
```

---

### Task 9: Handlers

**Files:**
- Create: `internal/handlers/auth_handler.go`
- Create: `internal/handlers/customer_handler.go`
- Create: `internal/handlers/user_handler.go`
- Create: `internal/handlers/role_handler.go`
- Create: `internal/handlers/permission_handler.go`
- Create: `internal/handlers/dashboard_handler.go`

- [ ] **Step 1: Create auth_handler.go**

```go
package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginInput struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) LoginAdmin(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	tokens, err := h.authService.LoginAdmin(input.Login, input.Password)
	if err != nil {
		return utils.Error(c, 401, "INVALID_CREDENTIALS", err.Error())
	}

	return utils.Success(c, tokens, "Login successful")
}

func (h *AuthHandler) LoginCustomer(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	tokens, err := h.authService.LoginCustomer(input.Login, input.Password)
	if err != nil {
		return utils.Error(c, 401, "INVALID_CREDENTIALS", err.Error())
	}

	return utils.Success(c, tokens, "Login successful")
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var input RefreshInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	accessToken, err := h.authService.Refresh(input.RefreshToken)
	if err != nil {
		return utils.Error(c, 401, "UNAUTHORIZED", err.Error())
	}

	return utils.Success(c, fiber.Map{"access_token": accessToken}, "Token refreshed")
}
```

- [ ] **Step 2: Create customer_handler.go**

```go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CustomerHandler struct {
	service *services.CustomerService
}

func NewCustomerHandler(service *services.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	customers, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch customers")
	}

	return utils.SuccessWithPagination(c, customers, page, limit, total)
}

func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	customer, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Customer not found")
	}

	return utils.Success(c, customer, "")
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var input services.CreateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	customer, err := h.service.Create(input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Customer created")
}

func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	customer, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Customer updated")
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Customer deleted")
}

func (h *CustomerHandler) GetProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid user ID")
	}

	customer, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "Customer not found")
	}

	return utils.Success(c, customer, "")
}

func (h *CustomerHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid user ID")
	}

	var input services.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	customer, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, customer, "Profile updated")
}
```

- [ ] **Step 3: Create user_handler.go**

```go
package handlers

import (
	"strconv"

	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	users, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch users")
	}

	return utils.SuccessWithPagination(c, users, page, limit, total)
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	user, err := h.service.GetByID(id)
	if err != nil {
		return utils.Error(c, 404, "NOT_FOUND", "User not found")
	}

	return utils.Success(c, user, "")
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var input services.CreateUserInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	user, err := h.service.Create(input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, user, "User created")
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	user, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, user, "User updated")
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "User deleted")
}
```

- [ ] **Step 4: Create role_handler.go**

```go
package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RoleHandler struct {
	service *services.RoleService
}

func NewRoleHandler(service *services.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) GetAll(c *fiber.Ctx) error {
	roles, err := h.service.GetAll()
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch roles")
	}
	return utils.Success(c, roles, "")
}

func (h *RoleHandler) Create(c *fiber.Ctx) error {
	var input services.CreateRoleInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	if errs := utils.Validate(input); errs != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "VALIDATION_ERROR", "message": errs}})
	}

	role, err := h.service.Create(input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, role, "Role created")
}

func (h *RoleHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	var input services.UpdateRoleInput
	if err := c.BodyParser(&input); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid request body")
	}

	role, err := h.service.Update(id, input)
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, role, "Role updated")
}

func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", "Invalid ID")
	}

	if err := h.service.Delete(id); err != nil {
		return utils.Error(c, 400, "VALIDATION_ERROR", err.Error())
	}

	return utils.Success(c, nil, "Role deleted")
}
```

- [ ] **Step 5: Create permission_handler.go**

```go
package handlers

import (
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type PermissionHandler struct {
	repo *repositories.PermissionRepository
}

func NewPermissionHandler(repo *repositories.PermissionRepository) *PermissionHandler {
	return &PermissionHandler{repo: repo}
}

func (h *PermissionHandler) GetAll(c *fiber.Ctx) error {
	permissions, err := h.repo.FindAll()
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch permissions")
	}
	return utils.Success(c, permissions, "")
}
```

- [ ] **Step 6: Create dashboard_handler.go**

```go
package handlers

import (
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	service *services.DashboardService
}

func NewDashboardHandler(service *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return utils.Error(c, 500, "INTERNAL_ERROR", "Failed to fetch stats")
	}
	return utils.Success(c, stats, "")
}
```

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/
git commit -m "feat: add handlers (auth, customer, user, role, permission, dashboard)"
```

---

### Task 10: Main Entry Point & Routes

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write main.go with routes and seed data**

```go
package main

import (
	"log"

	"go-fiber/internal/config"
	"go-fiber/internal/database"
	"go-fiber/internal/handlers"
	"go-fiber/internal/middleware"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg)
	database.Migrate(db)

	seedData(db, cfg)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return utils.Error(c, 500, "INTERNAL_ERROR", err.Error())
		},
	})

	app.Use(logger.New())
	app.Use(recover.New())

	// Repositories
	customerRepo := repositories.NewCustomerRepository(db)
	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRoleRepository(db)
	permissionRepo := repositories.NewPermissionRepository(db)

	// Services
	authService := services.NewAuthService(cfg, userRepo, customerRepo)
	customerService := services.NewCustomerService(customerRepo)
	userService := services.NewUserService(userRepo, roleRepo)
	roleService := services.NewRoleService(roleRepo, permissionRepo)
	dashboardService := services.NewDashboardService(db)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	customerHandler := handlers.NewCustomerHandler(customerService)
	userHandler := handlers.NewUserHandler(userService)
	roleHandler := handlers.NewRoleHandler(roleService)
	permissionHandler := handlers.NewPermissionHandler(permissionRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	api := app.Group("/api/v1")

	// Public routes
	auth := api.Group("/auth")
	auth.Post("/customer/login", authHandler.LoginCustomer)
	auth.Post("/admin/login", authHandler.LoginAdmin)
	auth.Post("/refresh", authHandler.Refresh)

	// Customer routes
	customer := api.Group("/customer", middleware.JWTAuth(cfg))
	customer.Get("/profile", customerHandler.GetProfile)
	customer.Put("/profile", customerHandler.UpdateProfile)

	// Admin routes
	admin := api.Group("/admin", middleware.JWTAuth(cfg))

	admin.Get("/dashboard/stats", middleware.RequirePermission(userRepo, "dashboard:read"), dashboardHandler.GetStats)

	// Customer management
	admin.Get("/customers", middleware.RequirePermission(userRepo, "customer:read"), customerHandler.GetAll)
	admin.Get("/customers/:id", middleware.RequirePermission(userRepo, "customer:read"), customerHandler.GetByID)
	admin.Post("/customers", middleware.RequirePermission(userRepo, "customer:write"), customerHandler.Create)
	admin.Put("/customers/:id", middleware.RequirePermission(userRepo, "customer:write"), customerHandler.Update)
	admin.Delete("/customers/:id", middleware.RequirePermission(userRepo, "customer:delete"), customerHandler.Delete)

	// User management
	admin.Get("/users", middleware.RequirePermission(userRepo, "user:read"), userHandler.GetAll)
	admin.Get("/users/:id", middleware.RequirePermission(userRepo, "user:read"), userHandler.GetByID)
	admin.Post("/users", middleware.RequirePermission(userRepo, "user:write"), userHandler.Create)
	admin.Put("/users/:id", middleware.RequirePermission(userRepo, "user:write"), userHandler.Update)
	admin.Delete("/users/:id", middleware.RequirePermission(userRepo, "user:delete"), userHandler.Delete)

	// Role management
	admin.Get("/roles", middleware.RequirePermission(userRepo, "role:read"), roleHandler.GetAll)
	admin.Post("/roles", middleware.RequirePermission(userRepo, "role:write"), roleHandler.Create)
	admin.Put("/roles/:id", middleware.RequirePermission(userRepo, "role:write"), roleHandler.Update)
	admin.Delete("/roles/:id", middleware.RequirePermission(userRepo, "role:delete"), roleHandler.Delete)

	// Permission management
	admin.Get("/permissions", middleware.RequirePermission(userRepo, "permission:read"), permissionHandler.GetAll)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	log.Fatal(app.Listen(":" + cfg.ServerPort))
}

func seedData(db *gorm.DB, cfg *config.Config) {
	var count int64
	db.Model(&models.Role{}).Count(&count)
	if count > 0 {
		return
	}

	permissions := []models.Permission{
		{Name: "customer:read", Description: "View customers"},
		{Name: "customer:write", Description: "Create/update customers"},
		{Name: "customer:delete", Description: "Delete customers"},
		{Name: "user:read", Description: "View users"},
		{Name: "user:write", Description: "Create/update users"},
		{Name: "user:delete", Description: "Delete users"},
		{Name: "role:read", Description: "View roles"},
		{Name: "role:write", Description: "Create/update roles"},
		{Name: "role:delete", Description: "Delete roles"},
		{Name: "permission:read", Description: "View permissions"},
		{Name: "permission:write", Description: "Create permissions"},
		{Name: "dashboard:read", Description: "View dashboard"},
	}

	for i := range permissions {
		db.Create(&permissions[i])
	}

	superAdmin := models.Role{
		ID:          uuid.New(),
		Name:        "super_admin",
		Description: "Full access",
		Permissions: permissions,
	}
	db.Create(&superAdmin)

	editor := models.Role{
		ID:          uuid.New(),
		Name:        "editor",
		Description: "Edit customers, view dashboard",
	}
	db.Create(&editor)
	db.Model(&editor).Association("Permissions").Append(
		[]models.Permission{permissions[0], permissions[1], permissions[11]},
	)

	viewer := models.Role{
		ID:          uuid.New(),
		Name:        "viewer",
		Description: "Read-only access",
	}
	db.Create(&viewer)
	db.Model(&viewer).Association("Permissions").Append(
		[]models.Permission{permissions[0], permissions[11]},
	)

	hashedPassword, _ := utils.HashPassword(cfg.AdminPassword)
	adminUser := models.User{
		Email:    &cfg.AdminEmail,
		Password: hashedPassword,
		Name:     "Super Admin",
		RoleID:   superAdmin.ID,
		Status:   "active",
	}
	if cfg.AdminPhone != "" {
		adminUser.PhoneNumber = &cfg.AdminPhone
	}
	db.Create(&adminUser)

	log.Println("Seed data created successfully")
}
```

- [ ] **Step 2: Fix import in main.go — add gorm import**

The import should include:
```go
import (
	"log"

	"go-fiber/internal/config"
	"go-fiber/internal/database"
	"go-fiber/internal/handlers"
	"go-fiber/internal/middleware"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"gorm.io/gorm"
)
```

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add main entry point with routes and seed data"
```

---

### Task 11: Final Verification

- [ ] **Step 1: Run go vet**

```bash
go vet ./...
```

- [ ] **Step 2: Verify all imports resolve**

```bash
go mod tidy
```

- [ ] **Step 3: Final build check**

```bash
go build ./cmd/server/
```

- [ ] **Step 4: Commit final cleanup**

```bash
git add .
git commit -m "chore: final cleanup and verification"
```

---

## Summary

| Task | Description |
|---|---|
| 1 | Project setup & dependencies |
| 2 | Config loader |
| 3 | Utils (response, password, validator) |
| 4 | Models (customer, user, role, permission) |
| 5 | Database connection & migration |
| 6 | Repositories |
| 7 | Services |
| 8 | Middleware (JWT auth, RBAC) |
| 9 | Handlers |
| 10 | Main entry point & routes |
| 11 | Final verification |
