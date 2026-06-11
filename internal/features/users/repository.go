package users

import (
	"strings"

	"golang-dashboard/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) Available() bool {
	return r.db != nil
}

func (r Repository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r Repository) Find(id uint) (models.User, error) {
	user := models.User{}
	err := r.db.First(&user, id).Error
	return user, err
}

func (r Repository) Save(user *models.User) error {
	return r.db.Save(user).Error
}

func (r Repository) Disable(id uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r Repository) List(filter ListFilter) ([]models.User, int64, error) {
	query := r.db.Model(&models.User{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	users := []models.User{}
	err := query.
		Order("created_at desc, id desc").
		Limit(filter.PerPage).
		Offset((filter.Page - 1) * filter.PerPage).
		Find(&users).Error
	return users, total, err
}

func (r Repository) IdentifierExists(role, email, opsID string, exceptID uint) (bool, error) {
	query := r.db.Model(&models.User{})
	if exceptID > 0 {
		query = query.Where("id <> ?", exceptID)
	}
	var count int64
	if isFTERole(role) {
		err := query.Where("LOWER(COALESCE(email, '')) = LOWER(?)", strings.TrimSpace(email)).Count(&count).Error
		return count > 0, err
	}
	err := query.Where("LOWER(COALESCE(ops_id, '')) = LOWER(?)", strings.TrimSpace(opsID)).Count(&count).Error
	return count > 0, err
}
