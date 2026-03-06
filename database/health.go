package database

import "context"

// DBHealthChecker implements core.HealthChecker for a GORM database connection.
// Register it with app.RegisterHealthChecker(database.NewHealthChecker(db))
// to expose the database status in GET /health.
type DBHealthChecker struct {
	instance *DBinstance
}

// NewHealthChecker returns a HealthChecker that pings the database.
func NewHealthChecker(instance *DBinstance) *DBHealthChecker {
	return &DBHealthChecker{instance: instance}
}

// Name returns the key used in the /health response (e.g. "database": "UP").
func (h *DBHealthChecker) Name() string {
	return "database"
}

// Check pings the database. Returns a non-nil error if the connection is unhealthy.
func (h *DBHealthChecker) Check(ctx context.Context) error {
	return h.instance.SQLDB().PingContext(ctx)
}
