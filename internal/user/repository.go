package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateUser = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const createTable = `
CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now()
);`
	_, err := pool.Exec(ctx, createTable)
	return err
}

func (r *Repository) Create(ctx context.Context, username, passwordHash string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username`, username, passwordHash).
		Scan(&u.ID, &u.Username)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrDuplicateUser
		}
		return User{}, err
	}
	return u, nil
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (User, string, error) {
	var u User
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT id, username, password_hash FROM users WHERE username=$1`, username).
		Scan(&u.ID, &u.Username, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, "", ErrUserNotFound
		}
		return User{}, "", err
	}
	return u, hash, nil
}

func (r *Repository) FindByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `SELECT id, username FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}
