package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/auth/internal/domain"
)

func newUUID() string {
	return uuid.New().String()
}

type AuthRepository interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetUserByEmailOrPhone(ctx context.Context, email, phone *string) (*domain.User, error)
	CreateUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	UpdateUser(ctx context.Context, userID string, input domain.UpdateUserInput) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
	DeleteUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, filter domain.ListUsersFilter) ([]*domain.User, int64, error)
	SetVerificationCode(ctx context.Context, userID, code string) error
	VerifyUser(ctx context.Context, userID string) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByIdentifier routes to email lookup if the identifier contains '@', phone otherwise.
func (r *authRepository) GetUserByEmailOrPhone(ctx context.Context, email, phone *string) (*domain.User, error) {
	var user domain.User
	query := r.db.Model(&domain.User{})
	if email != nil {
		query = query.Where("email = ?", *email)
	}
	if phone != nil {
		query = query.Or("phone = ?", *phone)
	}
	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email or phone: %v", err)
	}
	return &user, nil
}

func (r *authRepository) CreateUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	role := input.Role
	if role == "" {
		role = domain.RoleUser
	}
	user := &domain.User{
		ID:         newUUID(),
		Email:      input.Email,
		Phone:      input.Phone,
		Password:   input.Password,
		FirstName:  input.FirstName,
		LastName:   input.LastName,
		Role:       role,
		BusinessID: input.BusinessID,
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *authRepository) UpdateUser(ctx context.Context, userID string, input domain.UpdateUserInput) (*domain.User, error) {
	updates := map[string]interface{}{}
	if input.FirstName != "" {
		updates["first_name"] = input.FirstName
	}
	if input.LastName != "" {
		updates["last_name"] = input.LastName
	}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	if input.Address != "" {
		updates["address"] = input.Address
	}
	if input.City != "" {
		updates["city"] = input.City
	}
	if input.Country != "" {
		updates["country"] = input.Country
	}
	if input.Role != "" {
		updates["role"] = input.Role
	}
	if input.BusinessID != "" {
		updates["business_id"] = input.BusinessID
	}

	result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("user not found")
	}
	return r.GetUserByID(ctx, userID)
}

func (r *authRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("password", hashedPassword)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *authRepository) DeleteUser(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).Where("id = ?", userID).Delete(&domain.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *authRepository) ListUsers(ctx context.Context, filter domain.ListUsersFilter) ([]*domain.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.User{})
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.BusinessID != "" {
		query = query.Where("business_id = ?", filter.BusinessID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	var users []*domain.User
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *authRepository) SetVerificationCode(ctx context.Context, userID, code string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		Update("verification_code", code).Error
}

func (r *authRepository) VerifyUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_verified":       true,
			"verification_code": "",
		}).Error
}
