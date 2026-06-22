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
