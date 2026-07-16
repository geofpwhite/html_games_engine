// Package pgstore provides a PostgreSQL-backed implementation of the Store interface.
package pgstore

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/geofpwhite/html_games_engine/accounts/db"
	"github.com/geofpwhite/html_games_engine/accounts/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type postgresStore struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func NewStore() store.Store {
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	return &postgresStore{
		db:      conn,
		queries: db.New(conn),
	}
}

// Register creates a new user with the given username and password, returning the new user's ID.
func (s *postgresStore) Register(ctx context.Context, username, password string) (int32, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	return s.queries.CreateUser(ctx, db.CreateUserParams{
		Username: username,
		Password: string(hashedPwd),
	})
}

// Login authenticates a user by username and password, returning the user's ID on success.
func (s *postgresStore) Login(ctx context.Context, username, password string) (int32, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return 0, errors.New("invalid username or password")
	}
	return user.Userid, nil
}

// GetUser retrieves a user by their ID.
func (s *postgresStore) GetUser(ctx context.Context, userID int32) (store.User, error) {
	user, err := s.queries.GetUser(ctx, userID)
	if err != nil {
		return store.User{}, err
	}
	return store.User{ID: user.Userid, Username: user.Username}, nil
}

// GetUserByUsername retrieves a user by their username.
func (s *postgresStore) GetUserByUsername(ctx context.Context, username string) (store.User, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return store.User{}, err
	}
	return store.User{ID: user.Userid, Username: user.Username}, nil
}

// UpdateUserPassword updates the password for the given user.
func (s *postgresStore) UpdateUserPassword(ctx context.Context, userID int32, newPassword string) error {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		Userid:   userID,
		Password: string(hashedPwd),
	})
}

// DeleteUser deletes a user by their ID.
func (s *postgresStore) DeleteUser(ctx context.Context, userID int32) error {
	return s.queries.DeleteUser(ctx, userID)
}

// GetUsernames retrieves all usernames.
func (s *postgresStore) GetUsernames(ctx context.Context) ([]string, error) {
	return s.queries.GetUsernames(ctx)
}