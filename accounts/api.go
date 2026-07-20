// Package accounts implements user registration, login, presence tracking,
// and game-challenge signaling for the engine.
package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	IDGenerator "github.com/geofpwhite/html_games_engine/IDGenerator"
	"github.com/geofpwhite/html_games_engine/accounts/cache"
	"github.com/geofpwhite/html_games_engine/accounts/store"

	"github.com/gorilla/websocket"
)

const (
	sessionCookieName = "session_key"
	sessionMaxAge     = 20 * time.Minute
	cleanupInterval   = time.Minute
)

// newGamePaths maps a challengeable game type to the path of that game's own
// "new_game" route, so accepting a challenge creates the game the exact same
// way the game's own package already does - accounts never builds games itself.
var newGamePaths = map[string]string{
	"connect4":       "/connect4/new_game",
	"tictactoe":      "/tictactoe/new_game",
	"connectthedots": "/connect-the-dots/new_game",
}

type challenge struct {
	ID           string
	FromUserID   int32
	FromUsername string
	ToUserID     int32
	GameType     string
}

type accountsAPI struct {
	mux   *http.ServeMux
	store store.Store
	cache cache.Cache

	connsMu sync.Mutex
	conns   map[int32]*websocket.Conn

	challengesMu sync.Mutex
	challenges   map[string]*challenge
}

// AccountRoutes registers the account, presence, and challenge HTTP/websocket routes.
func AccountRoutes(
	r *http.ServeMux,
	tmpl *template.Template,
	upgrader *websocket.Upgrader,
	userStore store.Store,
	userCache cache.Cache,
) {
	a := &accountsAPI{
		mux:        r,
		store:      userStore,
		cache:      userCache,
		conns:      make(map[int32]*websocket.Conn),
		challenges: make(map[string]*challenge),
	}

	for path, page := range map[string]string{
		"GET /login":           "login.go.tmpl",
		"GET /register":        "register.go.tmpl",
		"GET /logout":          "logout.go.tmpl",
		"GET /change-password": "change_password.go.tmpl",
		"GET /challenge":       "challenge.go.tmpl",
	} {
		r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			if err := tmpl.ExecuteTemplate(w, page, nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}

	r.HandleFunc("POST /accounts/register", a.registerHandler)
	r.HandleFunc("POST /accounts/login", a.loginHandler)
	r.HandleFunc("POST /accounts/logout", a.authorized(a.logoutHandler))
	r.HandleFunc("POST /accounts/password", a.authorized(a.changePasswordHandler))
	r.HandleFunc("GET /accounts/online", a.authorized(a.onlineHandler))
	r.HandleFunc("GET /accounts/ws", a.authorized(a.wsHandler(upgrader)))
	r.HandleFunc("POST /accounts/challenge", a.authorized(a.challengeHandler))
	r.HandleFunc("POST /accounts/challenge/{challengeID}/accept", a.authorized(a.acceptChallengeHandler))
	r.HandleFunc("POST /accounts/challenge/{challengeID}/decline", a.authorized(a.declineChallengeHandler))

	go a.cleanupLoop(cleanupInterval, sessionMaxAge)
}

// authorized wraps a handler so it only runs for requests carrying a valid session cookie,
// passing the resolved user ID through directly instead of via the request context.
func (a *accountsAPI) authorized(next func(w http.ResponseWriter, r *http.Request, userID int32)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil {
			slog.Debug("unauthorized", "err:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := a.cache.GetSession(c.Value)
		if err != nil {
			slog.Debug("unauthorized", "err:", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r, userID)
	}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *accountsAPI) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}
	id, err := a.store.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "error registering user", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(struct {
		UserID int32 `json:"userID"`
	}{UserID: id}); err != nil {
		slog.Error("error writing register response", "error", err)
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *accountsAPI) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}
	userID, err := a.store.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	sessionKey := IDGenerator.GenerateID(32)
	if err := a.cache.SetSession(sessionKey, userID, req.Username); err != nil {
		http.Error(w, "error logging in", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionKey,
		Path:     "/accounts/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		UserID   int32  `json:"userID"`
		Username string `json:"username"`
	}{UserID: userID, Username: req.Username}); err != nil {
		slog.Error("error writing login response", "error", err)
	}
}

func (a *accountsAPI) logoutHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		if err := a.cache.DeleteSession(c.Value); err != nil {
			slog.Error("error deleting session", "error", err)
		}
	}
	a.closeConn(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/accounts/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusOK)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (a *accountsAPI) changePasswordHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "currentPassword and newPassword are required", http.StatusBadRequest)
		return
	}
	user, err := a.store.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := a.store.Login(r.Context(), user.Username, req.CurrentPassword); err != nil {
		http.Error(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}
	if err := a.store.UpdateUserPassword(r.Context(), userID, req.NewPassword); err != nil {
		http.Error(w, "error updating password", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *accountsAPI) onlineHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	online, err := a.cache.ListOnline()
	if err != nil {
		http.Error(w, "error listing online users", http.StatusInternalServerError)
		return
	}
	others := make([]cache.OnlineUser, 0, len(online))
	for _, u := range online {
		if u.UserID != userID {
			others = append(others, u)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(others); err != nil {
		slog.Error("error writing online response", "error", err)
	}
}

// wsHandler upgrades the connection and keeps it registered so challenge notifications
// can be pushed to this user in real time. The connection carries no application
// messages of its own; a closed/broken read loop is how we detect the user leaving.
func (a *accountsAPI) wsHandler(upgrader *websocket.Upgrader) func(w http.ResponseWriter, r *http.Request, userID int32) {
	return func(w http.ResponseWriter, r *http.Request, userID int32) {
		conn, err := upgrader.Upgrade(w, r, nil)
		slog.Log(context.Background(), slog.LevelDebug, "wsHandler Called")
		if err != nil {
			return
		}
		conn.WriteMessage(websocket.BinaryMessage, []byte("connected"))

		a.connsMu.Lock()
		if old, ok := a.conns[userID]; ok {
			old.Close() //nolint:errcheck // replaced by the new connection below
		}
		a.conns[userID] = conn
		a.connsMu.Unlock()

		defer a.removeConn(userID, conn)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}

func (a *accountsAPI) removeConn(userID int32, conn *websocket.Conn) {
	a.connsMu.Lock()
	if a.conns[userID] == conn {
		delete(a.conns, userID)
	}
	a.connsMu.Unlock()
	conn.Close() //nolint:errcheck // nothing to do if closing fails
}

func (a *accountsAPI) closeConn(userID int32) {
	a.connsMu.Lock()
	conn, ok := a.conns[userID]
	if ok {
		delete(a.conns, userID)
	}
	a.connsMu.Unlock()
	if ok {
		conn.Close() //nolint:errcheck // nothing to do if closing fails
	}
}

func (a *accountsAPI) push(userID int32, v any) {
	a.connsMu.Lock()
	conn, ok := a.conns[userID]
	a.connsMu.Unlock()
	if !ok {
		return
	}
	if err := conn.WriteJSON(v); err != nil {
		slog.Error("error pushing message to user", "userID", userID, "error", err)
	}
}

type challengeRequest struct {
	TargetUsername string `json:"targetUsername"`
	GameType       string `json:"gameType"`
}

type challengeMessage struct {
	Type         string `json:"type"`
	ChallengeID  string `json:"challengeID,omitempty"`
	FromUsername string `json:"fromUsername,omitempty"`
	GameType     string `json:"gameType"`
	GameID       string `json:"gameID,omitempty"`
}

func (a *accountsAPI) challengeHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	var req challengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUsername == "" || req.GameType == "" {
		http.Error(w, "targetUsername and gameType are required", http.StatusBadRequest)
		return
	}
	if _, ok := newGamePaths[req.GameType]; !ok {
		http.Error(w, "unknown game type", http.StatusBadRequest)
		return
	}
	from, err := a.store.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	target, err := a.store.GetUserByUsername(r.Context(), req.TargetUsername)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if target.ID == userID {
		http.Error(w, "cannot challenge yourself", http.StatusBadRequest)
		return
	}
	online, err := a.cache.IsOnline(target.ID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !online {
		http.Error(w, "that user is not online", http.StatusConflict)
		return
	}

	ch := &challenge{
		ID:           IDGenerator.GenerateID(16),
		FromUserID:   userID,
		FromUsername: from.Username,
		ToUserID:     target.ID,
		GameType:     req.GameType,
	}
	a.challengesMu.Lock()
	a.challenges[ch.ID] = ch
	a.challengesMu.Unlock()

	a.push(target.ID, challengeMessage{
		Type:         "challenge",
		ChallengeID:  ch.ID,
		FromUsername: from.Username,
		GameType:     ch.GameType,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		ChallengeID string `json:"challengeID"`
	}{ChallengeID: ch.ID}); err != nil {
		slog.Error("error writing challenge response", "error", err)
	}
}

func (a *accountsAPI) acceptChallengeHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	ch, ok := a.takeChallenge(r.PathValue("challengeID"), userID)
	if !ok {
		http.Error(w, "challenge not found", http.StatusNotFound)
		return
	}
	gameID, err := a.createGame(ch.GameType)
	if err != nil {
		http.Error(w, "error creating game", http.StatusInternalServerError)
		return
	}

	msg := challengeMessage{
		Type:     "challenge_accepted",
		GameType: ch.GameType,
		GameID:   gameID,
	}
	a.push(ch.FromUserID, msg)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		slog.Error("error writing challenge accept response", "error", err)
	}
}

// createGame creates a new game of the given type by dispatching an in-process request
// to that game's own "new_game" route, so the game is created exactly as it would be if
// a player had clicked "New Game" themselves - accounts never constructs games itself.
func (a *accountsAPI) createGame(gameType string) (string, error) {
	path, ok := newGamePaths[gameType]
	if !ok {
		return "", fmt.Errorf("unknown game type %q", gameType)
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		return "", fmt.Errorf("new_game route for %q returned status %d", gameType, rec.Code)
	}

	var withGameID struct {
		GameID string `json:"gameID"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &withGameID); err == nil && withGameID.GameID != "" {
		return withGameID.GameID, nil
	}
	var bare string
	if err := json.Unmarshal(rec.Body.Bytes(), &bare); err == nil && bare != "" {
		return bare, nil
	}
	return "", fmt.Errorf("could not parse game ID from new_game response for %q", gameType)
}

func (a *accountsAPI) declineChallengeHandler(w http.ResponseWriter, r *http.Request, userID int32) {
	ch, ok := a.takeChallenge(r.PathValue("challengeID"), userID)
	if !ok {
		http.Error(w, "challenge not found", http.StatusNotFound)
		return
	}
	a.push(ch.FromUserID, challengeMessage{
		Type:     "challenge_declined",
		GameType: ch.GameType,
	})
	w.WriteHeader(http.StatusOK)
}

// takeChallenge removes and returns a pending challenge, but only if it exists and
// was addressed to userID - this doubles as the authorization check for accept/decline.
func (a *accountsAPI) takeChallenge(challengeID string, userID int32) (*challenge, bool) {
	a.challengesMu.Lock()
	defer a.challengesMu.Unlock()
	ch, ok := a.challenges[challengeID]
	if !ok || ch.ToUserID != userID {
		return nil, false
	}
	delete(a.challenges, challengeID)
	return ch, true
}

// cleanupLoop periodically logs out users who have been inactive for longer than maxInactive.
func (a *accountsAPI) cleanupLoop(interval, maxInactive time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		stale, err := a.cache.PurgeInactive(maxInactive)
		if err != nil {
			slog.Error("error purging inactive sessions", "error", err)
			continue
		}
		for _, userID := range stale {
			a.closeConn(userID)
		}
	}
}
