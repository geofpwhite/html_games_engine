// Package gamesession records how long a logged-in user spends playing each game,
// keyed off the same session cookie the accounts package uses for authentication.
package gamesession

import (
	"context"
	"net/http"

	"github.com/geofpwhite/html_games_engine/accounts/cache"
	"github.com/geofpwhite/html_games_engine/accounts/store"
)

// CookieName is the session cookie set by accounts on login.
const CookieName = "session_key"

// Tracker starts and ends GameSessions rows for logged-in players.
type Tracker struct {
	store store.Store
	cache cache.Cache
}

// New builds a Tracker backed by the given store and cache.
func New(s store.Store, c cache.Cache) *Tracker {
	return &Tracker{store: s, cache: c}
}

// Start records that the player behind r's session cookie has begun playing gameType.
// If r carries no valid session (the player isn't logged in), it does nothing and
// reports ok=false - anonymous play is allowed but isn't tracked.
func (t *Tracker) Start(ctx context.Context, r *http.Request, gameType string) (sessionID int32, ok bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return 0, false
	}
	userID, err := t.cache.GetSession(c.Value)
	if err != nil {
		return 0, false
	}
	sessionID, err = t.store.StartGameSession(ctx, userID, gameType)
	if err != nil {
		return 0, false
	}
	return sessionID, true
}

// End records that sessionID has ended.
func (t *Tracker) End(ctx context.Context, sessionID int32) {
	_ = t.store.EndGameSession(ctx, sessionID)
}
