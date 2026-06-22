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

func (r *CustomerRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Customer{}).Count(&count).Error
	return count, err
}

func (r *CustomerRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Customer{}).Where("status = ?", status).Count(&count).Error
	return count, err
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
