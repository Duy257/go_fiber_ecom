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
