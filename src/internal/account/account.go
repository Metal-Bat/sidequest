package account

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrExists = errors.New("that username already exists")
var ErrInvalid = errors.New("use 3-32 letters, digits or underscores for the username and 10-72 bytes for the password")
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func Create(ctx context.Context, pool *pgxpool.Pool, username, password string, admin bool) error {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) || len(password) < 10 || len(password) > 72 {
		return ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `INSERT INTO users(username,password_hash,is_admin) VALUES($1,$2,$3)`, username, string(hash), admin)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return ErrExists
	}
	return err
}
