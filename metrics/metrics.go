// Package metrics defines the Prometheus metrics exported by the engine
// at GET /metrics, and a small helper to keep the periodic gauges fresh.
package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RegisteredUsers is the current total number of accounts in the store.
	RegisteredUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "html_games_registered_users",
		Help: "Total number of registered user accounts.",
	})

	// OnlineUsers is the current number of users with an active session.
	OnlineUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "html_games_online_users",
		Help: "Current number of users with an active session.",
	})

	// GamesPlayed counts every game created, labeled by game type.
	GamesPlayed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "html_games_played_total",
		Help: "Total number of games created, labeled by game type.",
	}, []string{"game"})

	// Visits counts page views, labeled by path.
	Visits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "html_games_visits_total",
		Help: "Total number of page visits, labeled by path.",
	}, []string{"path"})
)

// Handler returns the HTTP handler that serves metrics in the Prometheus text format.
func Handler() http.Handler {
	return promhttp.Handler()
}

// UserCounts is the minimal data source needed to keep the gauges above fresh.
type UserCounts interface {
	CountUsers(ctx context.Context) (int, error)
	CountOnline() (int, error)
}

// PollUserCounts periodically refreshes RegisteredUsers and OnlineUsers until ctx is done.
func PollUserCounts(ctx context.Context, counts UserCounts, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	refresh := func() {
		if n, err := counts.CountUsers(ctx); err != nil {
			slog.Error("metrics: error counting registered users", "error", err)
		} else {
			RegisteredUsers.Set(float64(n))
		}
		if n, err := counts.CountOnline(); err != nil {
			slog.Error("metrics: error counting online users", "error", err)
		} else {
			OnlineUsers.Set(float64(n))
		}
	}
	refresh()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}

// VisitMiddleware counts each GET request to an HTML page (as opposed to JSON/websocket
// API calls) so the dashboard can show site traffic.
func VisitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			Visits.WithLabelValues(r.URL.Path).Inc()
		}
		next.ServeHTTP(w, r)
	})
}
