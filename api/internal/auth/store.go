package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type Store interface {
	Register(ctx context.Context, name string, email string, password string) (User, error)
	Login(ctx context.Context, email string, password string) (User, error)
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Register(ctx context.Context, name string, email string, password string) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || email == "" || strings.TrimSpace(password) == "" {
		return User{}, errors.New("name, email and password are required")
	}

	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}

	_, err = s.pool.Exec(ctx,
		`insert into users (name, email, password_hash) values ($1, $2, $3)`,
		name,
		email,
		hash,
	)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return User{Name: name, Email: email, Token: makeToken(email)}, nil
}

func (s *PostgresStore) Login(ctx context.Context, email string, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.TrimSpace(password) == "" {
		return User{}, errors.New("email and password are required")
	}

	var user User
	var storedHash string
	err := s.pool.QueryRow(ctx,
		`select name, email, password_hash from users where email = $1`,
		email,
	).Scan(&user.Name, &user.Email, &storedHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, errors.New("invalid email or password")
	}
	if err != nil {
		return User{}, fmt.Errorf("load user: %w", err)
	}
	if !verifyPassword(password, storedHash) {
		return User{}, errors.New("invalid email or password")
	}
	user.Token = makeToken(user.Email)
	return user, nil
}

type DisabledStore struct{}

func (DisabledStore) Register(context.Context, string, string, string) (User, error) {
	return User{}, errors.New("DATABASE_URL is required for register")
}

func (DisabledStore) Login(context.Context, string, string) (User, error) {
	return User{}, errors.New("DATABASE_URL is required for login")
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := passwordDigest(salt, password)
	return base64.RawStdEncoding.EncodeToString(salt) + ":" + base64.RawStdEncoding.EncodeToString(digest), nil
}

func verifyPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, ":")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := passwordDigest(salt, password)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func passwordDigest(salt []byte, password string) []byte {
	sum := sha256.Sum256(append(append([]byte{}, salt...), []byte(password)...))
	return sum[:]
}

func makeToken(email string) string {
	seed := fmt.Sprintf("%s:%d", email, time.Now().UnixNano())
	sum := sha256.Sum256([]byte(seed))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
