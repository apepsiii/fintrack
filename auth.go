package main

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
}

// Claims represents JWT claims
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret []byte
	jwtExpiry time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService() *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fintrack-dev-secret-key-2026" // Default for development
	}

	expiryHours := 24
	if envExpiry := os.Getenv("JWT_EXPIRY_HOURS"); envExpiry != "" {
		if hours, err := strconv.Atoi(envExpiry); err == nil {
			expiryHours = hours
		}
	}

	return &AuthService{
		jwtSecret: []byte(secret),
		jwtExpiry: time.Duration(expiryHours) * time.Hour,
	}
}

// HashPassword hashes a plain text password using bcrypt
func (a *AuthService) HashPassword(password string) (string, error) {
	cost := bcrypt.DefaultCost
	if envCost := os.Getenv("BCRYPT_COST"); envCost != "" {
		if c, err := strconv.Atoi(envCost); err == nil && c >= bcrypt.MinCost && c <= bcrypt.MaxCost {
			cost = c
		}
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword verifies a password against its hash
func (a *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a JWT token for a user
func (a *AuthService) GenerateToken(user *User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "fintrack",
			Subject:   strconv.Itoa(user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RegisterUser creates a new user account
func RegisterUser(name, email, password string) (*User, error) {
	// Check if user already exists
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("email already registered")
	}

	// Hash password
	authService := NewAuthService()
	passwordHash, err := authService.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Insert user
	result, err := db.Exec(`
		INSERT INTO users (name, email, password_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, name, email, passwordHash, time.Now())
	if err != nil {
		return nil, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        int(userID),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}, nil
}

// AuthenticateUser validates credentials and returns user if valid
func AuthenticateUser(email, password string) (*User, error) {
	var user User
	var passwordHash string

	err := db.QueryRow(`
		SELECT id, name, email, password_hash, COALESCE(avatar,''), created_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Name, &user.Email, &passwordHash, &user.Avatar, &user.CreatedAt)

	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	authService := NewAuthService()
	if !authService.CheckPassword(password, passwordHash) {
		return nil, errors.New("invalid email or password")
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(userID int) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT id, name, email, COALESCE(avatar,''), created_at
		FROM users WHERE id = ?
	`, userID).Scan(&user.ID, &user.Name, &user.Email, &user.Avatar, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
