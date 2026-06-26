package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"thugcorp.io/grocery/auth/internal/domain"
	"thugcorp.io/grocery/auth/internal/utils"
)

type AuthService interface {
	Signup(ctx context.Context, input domain.CreateUserInput) (*domain.User, bool, string, error)
	Login(ctx context.Context, email, phone, password string) (*domain.User, string, error)
	VerifyCode(ctx context.Context, userID, code string) (*domain.User, string, error)
	ResendCode(ctx context.Context, userID string) error
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID string, input domain.UpdateUserInput) (*domain.User, error)
	ForgotPassword(ctx context.Context, emailOrPhone string) error
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
	CreateUser(ctx context.Context, callerRole string, input domain.CreateUserInput) (*domain.User, error)
	ListUsers(ctx context.Context, filter domain.ListUsersFilter) ([]*domain.User, error)
	UpdateUser(ctx context.Context, callerRole, targetUserID string, input domain.UpdateUserInput) (*domain.User, error)
}

// tokenClaims mirrors the fields parsed by middleware.CustomClaims so tokens
// issued here can be validated by every downstream service.
type tokenClaims struct {
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	BusinessID string `json:"business_id"`
	jwt.RegisteredClaims
}

type authService struct {
	authRepository AuthRepository
	email          utils.EmailService
	sms            utils.SMSService
	privateKey     *rsa.PrivateKey
	tokenTTL       int
}

func NewAuthService(authRepository AuthRepository, sms utils.SMSService, email utils.EmailService, privateKey *rsa.PrivateKey, tokenTTL int) *authService {
	return &authService{
		authRepository: authRepository,
		email:          email,
		sms:            sms,
		privateKey:     privateKey,
		tokenTTL:       tokenTTL,
	}
}

// ---- Public flows ----

func (s *authService) Signup(ctx context.Context, input domain.CreateUserInput) (*domain.User, bool, string, error) {
	if input.Email == "" && input.Phone == "" {
		return nil, false, "", errors.New("email or phone number is required")
	}

	if input.Email != "" {
		existing, err := s.authRepository.GetUserByEmailOrPhone(ctx, &input.Email, nil)
		if err != nil {
			return nil, false, "", err
		}
		if existing != nil {
			return nil, false, "", errors.New("email already registered")
		}
	}

	if input.Phone != "" {
		existing, err := s.authRepository.GetUserByEmailOrPhone(ctx, nil, &input.Phone)
		if err != nil {
			return nil, false, "", err
		}
		if existing != nil {
			return nil, false, "", errors.New("phone already registered")
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, "", err
	}
	input.Password = string(hashed)
	input.Role = domain.RoleUser

	user, err := s.authRepository.CreateUser(ctx, input)
	if err != nil {
		return nil, false, "", err
	}

	code, err := generateVerificationCode()
	if err != nil {
		return nil, false, "", err
	}
	if err := s.authRepository.SetVerificationCode(ctx, user.ID, code); err != nil {
		return nil, false, "", err
	}

	verifyMethod := "email"
	if input.Email != "" {
		if err := s.email.SendVerificationCode(input.Email, code); err != nil {
			log.Printf("failed to send verification email to %s: %v", input.Email, err)
		}
	} else {
		verifyMethod = "sms"
		if err := s.sms.SendVerificationCode(input.Phone, code); err != nil {
			log.Printf("failed to send verification SMS to %s: %v", input.Phone, err)
		}
	}
	return user, true, verifyMethod, nil
}

func (s *authService) Login(ctx context.Context, email, phone, password string) (*domain.User, string, error) {
	if email == "" && phone == "" {
		return nil, "", errors.New("email or phone number is required")
	}

	user, err := s.authRepository.GetUserByEmailOrPhone(ctx, &email, &phone)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid password")
	}

	if !user.IsVerified {
		return user, "", errors.New("account not verified")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *authService) VerifyCode(ctx context.Context, userID, code string) (*domain.User, string, error) {
	user, err := s.authRepository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", errors.New("user not found")
	}
	if user.IsVerified {
		token, err := s.generateToken(user)
		return user, token, err
	}
	if user.VerificationCode != code {
		return nil, "", errors.New("invalid verification code")
	}

	if err := s.authRepository.VerifyUser(ctx, user.ID); err != nil {
		return nil, "", err
	}
	user.IsVerified = true

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *authService) ResendCode(ctx context.Context, userID string) error {
	user, err := s.authRepository.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.IsVerified {
		return errors.New("account already verified")
	}

	code, err := generateVerificationCode()
	if err != nil {
		return err
	}
	if err := s.authRepository.SetVerificationCode(ctx, user.ID, code); err != nil {
		return err
	}
	if user.Email != "" {
		if err := s.email.SendVerificationCode(user.Email, code); err != nil {
			log.Printf("failed to resend verification email to %s: %v", user.Email, err)
		}
	} else {
		if err := s.sms.SendVerificationCode(user.Phone, code); err != nil {
			log.Printf("failed to resend verification SMS to %s: %v", user.Phone, err)
		}
	}
	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, emailOrPhone string) error {
	if emailOrPhone == "" {
		return errors.New("email or phone number is required")
	}

	user, err := s.authRepository.GetUserByEmailOrPhone(ctx, &emailOrPhone, &emailOrPhone)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("user not found")
	}

	code, err := generateVerificationCode()
	if err != nil {
		return err
	}

	if err := s.authRepository.SetVerificationCode(ctx, user.ID, code); err != nil {
		return err
	}
	if user.Email != "" {
		if err := s.email.SendVerificationCode(user.Email, code); err != nil {
			log.Printf("failed to send verification email to %s: %v", user.Email, err)
		}
	} else {
		if err := s.sms.SendVerificationCode(user.Phone, code); err != nil {
			log.Printf("failed to send verification SMS to %s: %v", user.Phone, err)
		}
	}
	return nil
}

// ---- Self-service ----

func (s *authService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.authRepository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *authService) UpdateProfile(ctx context.Context, userID string, input domain.UpdateUserInput) (*domain.User, error) {
	// Users cannot escalate their own role or change their business affiliation.
	input.Role = ""
	input.BusinessID = ""
	return s.authRepository.UpdateUser(ctx, userID, input)
}

func (s *authService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.authRepository.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.authRepository.UpdatePassword(ctx, userID, string(hashed))
}

// ---- Admin operations (super-admin only for CreateUser) ----

func (s *authService) CreateUser(ctx context.Context, callerRole string, input domain.CreateUserInput) (*domain.User, error) {
	if callerRole != domain.RoleSuperAdmin {
		return nil, errors.New("only super-admin can create users")
	}
	if input.Role == "" {
		input.Role = domain.RoleUser
	}

	existing, err := s.authRepository.GetUserByEmailOrPhone(ctx, &input.Email, &input.Phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email or phone already registered")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	input.Password = string(hashed)

	user, err := s.authRepository.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}

	// Admin-created accounts skip email verification.
	if err := s.authRepository.VerifyUser(ctx, user.ID); err != nil {
		return nil, err
	}
	user.IsVerified = true
	return user, nil
}

func (s *authService) ListUsers(ctx context.Context, filter domain.ListUsersFilter) ([]*domain.User, error) {
	users, _, err := s.authRepository.ListUsers(ctx, filter)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *authService) UpdateUser(ctx context.Context, callerRole, targetUserID string, input domain.UpdateUserInput) (*domain.User, error) {
	if callerRole == domain.RoleSuperAdmin {
		return s.authRepository.UpdateUser(ctx, targetUserID, input)
	}
	// Admins can only assign a business — not change roles or other fields.
	if callerRole == domain.RoleAdmin && input.Role == "" {
		return s.authRepository.UpdateUser(ctx, targetUserID, domain.UpdateUserInput{BusinessID: input.BusinessID})
	}
	return nil, errors.New("forbidden: insufficient permissions to update this user")
}

// ---- Helpers ----

func (s *authService) generateToken(user *domain.User) (string, error) {
	ttl := s.tokenTTL
	if ttl <= 0 {
		ttl = 3600 // Default to 1 hour if not set
	}
	claims := tokenClaims{
		UserID:     user.ID,
		Role:       user.Role,
		BusinessID: user.BusinessID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func generateVerificationCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}
