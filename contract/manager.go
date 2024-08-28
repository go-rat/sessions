package contract

type Manager interface {
	// BuildSession constructs a new session with the given handler and session ID.
	BuildSession(name, driver string, sessionID ...string) (Session, error)
	// Extend extends the session manager with a custom driver.
	Extend(driver string, handler Driver) Manager
}
