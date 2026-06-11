package users

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang-dashboard/internal/events"
	"golang-dashboard/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) List(filter ListFilter) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	filter.Page = clamp(filter.Page, 1, 100000)
	filter.PerPage = clamp(filter.PerPage, 1, 100)
	users, total, err := s.repo.List(filter)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load users"}
	}
	return map[string]interface{}{
		"users":    users,
		"count":    len(users),
		"total":    total,
		"page":     filter.Page,
		"per_page": filter.PerPage,
	}, nil
}

func (s Service) Create(payload Payload) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	if !canManageRoles(payload.ActorRole) {
		return nil, AppError{Code: http.StatusForbidden, Message: "Only FTE Ops and FTE MM can add roles"}
	}

	user, err := s.userFromPayload(models.User{IsActive: true, FirstTimeLogin: true}, payload, 0)
	if err != nil {
		return nil, err
	}
	ensureUserUniqueID(&user)
	if err := s.repo.Create(&user); err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to create user"}
	}
	events.DefaultBus.Publish(userEvent(events.UserCreated, "create", user))
	return response(user), nil
}

func (s Service) Update(id uint, payload Payload) (models.User, error) {
	if !s.repo.Available() {
		return models.User{}, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	if !canManageRoles(payload.ActorRole) {
		return models.User{}, AppError{Code: http.StatusForbidden, Message: "Only FTE Ops and FTE MM can update users"}
	}
	user, err := s.repo.Find(id)
	if err != nil {
		return models.User{}, AppError{Code: http.StatusNotFound, Message: "User not found"}
	}
	user, err = s.userFromPayload(user, payload, id)
	if err != nil {
		return models.User{}, err
	}
	if err := s.repo.Save(&user); err != nil {
		return models.User{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to update user"}
	}
	events.DefaultBus.Publish(userEvent(events.UserUpdated, "update", user))
	return user, nil
}

func (s Service) Disable(id uint, payload Payload) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	if !canManageRoles(payload.ActorRole) {
		return nil, AppError{Code: http.StatusForbidden, Message: "Only FTE Ops and FTE MM can disable users"}
	}
	if err := s.repo.Disable(id); err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to disable user"}
	}
	events.DefaultBus.Publish(events.Event{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:        events.UserDisabled,
		OccurredAt:  time.Now(),
		Aggregate:   "user",
		AggregateID: id,
		Action:      "disable",
	})
	return map[string]interface{}{"id": id, "is_active": false}, nil
}

func (s Service) userFromPayload(user models.User, payload Payload, exceptID uint) (models.User, error) {
	name := strings.TrimSpace(payload.Name)
	role := normalizeRole(payload.Role)
	email := strings.TrimSpace(payload.Email)
	opsID := strings.TrimSpace(payload.OpsID)
	isFTE := isFTERole(role)
	if isFTE {
		opsID = ""
	} else {
		email = ""
	}
	if name == "" {
		return user, AppError{Code: http.StatusBadRequest, Message: "Name is required"}
	}
	if role == "" {
		return user, AppError{Code: http.StatusBadRequest, Message: "Choose a valid role"}
	}
	if isFTE && email == "" {
		return user, AppError{Code: http.StatusBadRequest, Message: "Email is required for FTE Ops and FTE MM"}
	}
	if !isFTE && opsID == "" {
		return user, AppError{Code: http.StatusBadRequest, Message: "Ops ID is required for Backroom roles"}
	}
	if exists, err := s.repo.IdentifierExists(role, email, opsID, exceptID); err != nil {
		return user, AppError{Code: http.StatusInternalServerError, Message: "Unable to validate user identifier"}
	} else if exists {
		return user, AppError{Code: http.StatusConflict, Message: "A user with this identifier already exists"}
	}

	user.Name = name
	user.Role = role
	user.IsFTE = isFTE
	if isFTE {
		user.Email = stringPtr(email)
		user.OpsID = nil
	} else {
		user.OpsID = stringPtr(opsID)
		user.Email = nil
	}
	if passwordHash, err := hashPassword(payload.Password); err != nil {
		return user, AppError{Code: http.StatusBadRequest, Message: "Password is invalid"}
	} else if passwordHash != "" {
		user.PasswordHash = stringPtr(passwordHash)
	}
	return user, nil
}

func userEvent(eventType, action string, user models.User) events.Event {
	return events.Event{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:        eventType,
		OccurredAt:  time.Now(),
		Aggregate:   "user",
		AggregateID: user.ID,
		Action:      action,
		Payload: map[string]interface{}{
			"name": user.Name,
			"role": user.Role,
		},
	}
}

func response(user models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"role":       user.Role,
		"role_label": roleLabel(user.Role),
		"email":      user.Email,
		"ops_id":     user.OpsID,
		"is_fte":     user.IsFTE,
		"is_active":  user.IsActive,
	}
}

func canManageRoles(role string) bool {
	role = normalizeRole(role)
	return role == "fte_ops" || role == "fte_mm"
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "fte_ops", "fte_mm", "ops_pic", "dock_officer":
		return strings.ToLower(strings.TrimSpace(role))
	case "doc_officer":
		return "dock_officer"
	default:
		return ""
	}
}

func isFTERole(role string) bool {
	return role == "fte_ops" || role == "fte_mm"
}

func ensureUserUniqueID(user *models.User) {
	if strings.TrimSpace(user.UniqueID) != "" {
		return
	}
	prefix := "BR"
	if isFTERole(user.Role) {
		prefix = "FTE"
	}
	user.UniqueID = prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func hashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func stringPtr(value string) *string {
	return &value
}

func roleLabel(role string) string {
	switch role {
	case "ops_pic":
		return "Ops PIC"
	case "fte_ops":
		return "FTE Ops"
	case "fte_mm":
		return "FTE MM"
	case "dock_officer", "doc_officer":
		return "Dock Officer"
	default:
		return role
	}
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
