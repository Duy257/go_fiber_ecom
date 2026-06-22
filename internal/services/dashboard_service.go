package services

import (
	"go-fiber/internal/repositories"
)

type DashboardService struct {
	customerRepo *repositories.CustomerRepository
	userRepo     *repositories.UserRepository
	roleRepo     *repositories.RoleRepository
}

func NewDashboardService(customerRepo *repositories.CustomerRepository, userRepo *repositories.UserRepository, roleRepo *repositories.RoleRepository) *DashboardService {
	return &DashboardService{
		customerRepo: customerRepo,
		userRepo:     userRepo,
		roleRepo:     roleRepo,
	}
}

type DashboardStats struct {
	TotalCustomers  int64 `json:"total_customers"`
	TotalUsers      int64 `json:"total_users"`
	TotalRoles      int64 `json:"total_roles"`
	ActiveCustomers int64 `json:"active_customers"`
}

func (s *DashboardService) GetStats() (*DashboardStats, error) {
	var stats DashboardStats
	var err error

	stats.TotalCustomers, err = s.customerRepo.Count()
	if err != nil {
		return nil, err
	}
	stats.ActiveCustomers, err = s.customerRepo.CountByStatus("active")
	if err != nil {
		return nil, err
	}
	stats.TotalUsers, err = s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	stats.TotalRoles, err = s.roleRepo.Count()
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
