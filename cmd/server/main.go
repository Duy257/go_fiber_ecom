package main

import (
	"log"
	"time"

	"go-fiber/internal/config"
	"go-fiber/internal/database"
	"go-fiber/internal/handlers"
	"go-fiber/internal/middleware"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/services"
	"go-fiber/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
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
	dashboardService := services.NewDashboardService(customerRepo, userRepo, roleRepo)

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

	authLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	auth.Post("/customer/register", authLimiter, authHandler.RegisterCustomer)
	auth.Post("/customer/login", authLimiter, authHandler.LoginCustomer)
	auth.Post("/admin/login", authLimiter, authHandler.LoginAdmin)
	auth.Post("/refresh", authHandler.Refresh)

	// Customer routes (self-service)
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
	admin.Post("/permissions", middleware.RequirePermission(userRepo, "permission:write"), permissionHandler.Create)

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

	permMap := make(map[string]models.Permission)
	for _, p := range permissions {
		permMap[p.Name] = p
	}

	editor := models.Role{
		ID:          uuid.New(),
		Name:        "editor",
		Description: "Edit customers, view dashboard",
	}
	db.Create(&editor)
	db.Model(&editor).Association("Permissions").Append(
		[]models.Permission{permMap["customer:read"], permMap["customer:write"], permMap["dashboard:read"]},
	)

	viewer := models.Role{
		ID:          uuid.New(),
		Name:        "viewer",
		Description: "Read-only access",
	}
	db.Create(&viewer)
	db.Model(&viewer).Association("Permissions").Append(
		[]models.Permission{permMap["customer:read"], permMap["dashboard:read"]},
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
