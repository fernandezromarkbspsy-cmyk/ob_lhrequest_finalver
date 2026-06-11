package auth

import (
	"strings"
	"time"

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

func (r Repository) FindLoginUser(loginType, email, opsID string) (models.User, error) {
	user := models.User{}
	query := r.db.Model(&models.User{}).Where("is_active = ?", true)
	switch loginType {
	case "fte":
		query = query.Where("LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ?", email, []string{"fte_ops", "fte_mm"})
	case "backroom":
		query = query.Where("LOWER(COALESCE(ops_id, '')) = LOWER(?) AND role IN ?", opsID, []string{"ops_pic", "dock_officer", "doc_officer"})
	}
	err := query.First(&user).Error
	return user, err
}

func (r Repository) CountActiveFTEByEmail(email string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).
		Where("LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ? AND is_active = ?", strings.TrimSpace(email), []string{"fte_ops", "fte_mm"}, true).
		Count(&count).Error
	return count, err
}

func (r Repository) CreateOTP(otp *models.UserOTP) error {
	return r.db.Create(otp).Error
}

func (r Repository) LatestValidOTP(email string, now time.Time) (models.UserOTP, error) {
	otp := models.UserOTP{}
	err := r.db.
		Where("LOWER(email) = LOWER(?) AND used_at IS NULL AND expires_at > ?", strings.TrimSpace(email), now).
		Order("created_at desc, id desc").
		First(&otp).Error
	return otp, err
}

func (r Repository) MarkOTPUsed(otp *models.UserOTP, now time.Time) error {
	return r.db.Model(otp).Update("used_at", &now).Error
}

func (r Repository) ActiveFTEByEmail(email string) (models.User, error) {
	user := models.User{}
	err := r.db.
		Where("LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ? AND is_active = ?", strings.TrimSpace(email), []string{"fte_ops", "fte_mm"}, true).
		First(&user).Error
	return user, err
}

func (r Repository) FindUser(id uint) (models.User, error) {
	user := models.User{}
	err := r.db.First(&user, id).Error
	return user, err
}

func (r Repository) SaveUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r Repository) EnsureUniqueID(user *models.User) error {
	return r.db.Model(user).Update("unique_id", user.UniqueID).Error
}
