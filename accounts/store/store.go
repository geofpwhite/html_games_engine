// Package store provides the Store interface for account data access.
package store

import "context"

// User is the public representation of a user, safe to hand back to callers.
type User struct {
	ID       int32
	Username string
}

// Store defines the interface for all account data access operations.
type Store interface {
	// Register creates a new user with the given username and password, returning the new user's ID.
	Register(ctx context.Context, username, password string) (int32, error)
	// Login authenticates a user by username and password, returning the user's ID on success.
	Login(ctx context.Context, username, password string) (int32, error)
	// GetUser retrieves a user by their ID.
	GetUser(ctx context.Context, userID int32) (User, error)
	// GetUserByUsername retrieves a user by their username.
	GetUserByUsername(ctx context.Context, username string) (User, error)
	// UpdateUserPassword updates the password for the given user.
	UpdateUserPassword(ctx context.Context, userID int32, newPassword string) error
	// DeleteUser deletes a user by their ID.
	DeleteUser(ctx context.Context, userID int32) error
	// GetUsernames retrieves all usernames.
	GetUsernames(ctx context.Context) ([]string, error)
}
