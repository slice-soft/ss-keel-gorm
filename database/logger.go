package database

// Logger is the logging interface expected by the database package.
// It is satisfied by *logger.Logger from ss-keel-core, which means
// this addon is designed to be used exclusively within the Keel ecosystem.
//
// To inject the logger from a Keel application:
//
//	database.New(database.Config{Logger: app.Logger(), ...})
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}
