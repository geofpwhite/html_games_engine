// Package cache defines the Cache interface for session and online-presence tracking.
package cache

import "time"

// OnlineUser identifies a currently logged-in user.
type OnlineUser struct {
	UserID   int32
	Username string
}

// Cache defines methods for session management and online-presence tracking.
type Cache interface {
	// SetSession stores a session key for a user and marks them online.
	SetSession(sessionKey string, userID int32, username string) error
	// GetSession retrieves the user ID associated with a session key and refreshes its activity.
	GetSession(sessionKey string) (int32, error)
	// DeleteSession removes a session key from the cache and marks the user offline.
	DeleteSession(sessionKey string) error
	// ListOnline returns every user currently marked online.
	ListOnline() ([]OnlineUser, error)
	// IsOnline reports whether the given user is currently online.
	IsOnline(userID int32) (bool, error)
	// PurgeInactive removes and logs out any user whose last activity is older than maxAge,
	// returning the IDs of the users that were logged out.
	PurgeInactive(maxAge time.Duration) ([]int32, error)
}
