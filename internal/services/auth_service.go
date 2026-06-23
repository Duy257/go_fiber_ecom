package services

import (
	"errors"
	"time"

	"go-fiber/internal/config"
	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	cfg          *config.Config
	userRepo     *repositories.UserRepository
	customerRepo *repositories.CustomerRepository
}

func NewAuthService(cfg *config.Config, userRepo *repositories.UserRepository, customerRepo *repositories.CustomerRepository) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, customerRepo: customerRepo}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *AuthService) RegisterCustomer(input CreateCustomerInput) (*models.Customer, *TokenPair, error) {
	if input.Email == "" && input.PhoneNumber == "" {
		return nil, nil, errors.New("email or phone_number is required")
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, nil, err
	}

	customer := &models.Customer{
		Password: hashedPassword,
		Name:     input.Name,
		Status:   "active",
	}

	if input.Email != "" {
		customer.Email = &input.Email
	}
	if input.PhoneNumber != "" {
		customer.PhoneNumber = &input.PhoneNumber
	}

	if err := s.customerRepo.Create(customer); err != nil {
		return nil, nil, err
	}

	tokens, err := s.generateTokenPair(customer.ID.String(), "customer")
	if err != nil {
		return nil, nil, err
	}

	return customer, tokens, nil
}

func (s *AuthService) LoginAdmin(login, password string) (*TokenPair, error) {
	user, err := s.userRepo.FindByEmailOrPhone(login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	return s.generateTokenPair(user.ID.String(), "admin")
}

func (s *AuthService) LoginCustomer(login, password string) (*TokenPair, error) {
	customer, err := s.customerRepo.FindByEmailOrPhone(login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, customer.Password) {
		return nil, errors.New("invalid credentials")
	}

	if customer.Status != "active" {
		return nil, errors.New("account is inactive")
	}

	return s.generateTokenPair(customer.ID.String(), "customer")
}

func (s *AuthService) Refresh(refreshToken string) (string, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		return "", errors.New("invalid token type")
	}

	sub, _ := claims["sub"].(string)
	roleType, _ := claims["role"].(string)

	return s.generateAccessToken(sub, roleType)
}

func (s *AuthService) generateTokenPair(sub, roleType string) (*TokenPair, error) {
	accessToken, err := s.generateAccessToken(sub, roleType)
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"type": "refresh",
		"role": roleType,
		"exp":  time.Now().Add(s.cfg.JWTRefreshTTL).Unix(),
	})

	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshStr,
	}, nil
}

func (s *AuthService) generateAccessToken(sub, roleType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"type": "access",
		"role": roleType,
		"exp":  time.Now().Add(s.cfg.JWTAccessTTL).Unix(),
	})

	return token.SignedString([]byte(s.cfg.JWTSecret))
}
