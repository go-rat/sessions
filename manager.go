package session

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-rat/securecookie"

	"github.com/go-rat/session/driver"
)

type ManagerOptions struct {
	// 32 bytes string to encrypt session data
	Key string
	// session lifetime in minutes
	Lifetime int
	// session garbage collection interval in minutes
	GcInterval int
}

type Manager struct {
	codec       securecookie.Codec
	lifetime    int
	gcInterval  int
	drivers     map[string]driver.Driver
	sessionPool sync.Pool
}

// NewManager creates a new session manager.
func NewManager(option *ManagerOptions) (*Manager, error) {
	codec, err := securecookie.New([]byte(option.Key), &securecookie.Options{
		MaxAge:     int64(option.Lifetime),
		Serializer: securecookie.GobEncoder{},
	})
	if err != nil {
		return nil, err
	}
	manager := &Manager{
		codec:      codec,
		lifetime:   option.Lifetime,
		gcInterval: option.GcInterval,
		drivers:    make(map[string]driver.Driver),
		sessionPool: sync.Pool{New: func() any {
			return &Session{
				attributes: make(map[string]any),
			}
		},
		},
	}
	manager.createDefaultDriver()
	return manager, nil
}

func (m *Manager) BuildSession(name, driver string, sessionID ...string) (*Session, error) {
	handler, err := m.driver(driver)
	if err != nil {
		return nil, err
	}

	session := m.AcquireSession()
	session.name = name
	session.codec = m.codec
	session.driver = handler
	if len(sessionID) > 0 {
		session.SetID(sessionID[0])
	} else {
		session.SetID("")
	}

	return session, nil
}

func (m *Manager) Extend(driver string, handler driver.Driver) *Manager {
	m.drivers[driver] = handler
	m.startGcTimer(m.drivers[driver])
	return m
}

func (m *Manager) AcquireSession() *Session {
	session := m.sessionPool.Get().(*Session)
	return session
}

func (m *Manager) ReleaseSession(session *Session) {
	session.reset()
	m.sessionPool.Put(session)
}

func (m *Manager) driver(name ...string) (driver.Driver, error) {
	var driverName string
	if len(name) > 0 {
		driverName = name[0]
	} else {
		driverName = "default"
	}

	if driverName == "" {
		return nil, fmt.Errorf("driver is not set")
	}

	if m.drivers[driverName] == nil {
		return nil, fmt.Errorf("driver [%s] not supported", driverName)
	}

	return m.drivers[driverName], nil
}

func (m *Manager) startGcTimer(driver driver.Driver) {
	ticker := time.NewTicker(time.Duration(m.gcInterval) * time.Minute)

	go func() {
		for range ticker.C {
			if err := driver.Gc(m.lifetime * 60); err != nil {
				log.Printf("session gc error: %v\n", err)
			}
		}
	}()
}

func (m *Manager) createDefaultDriver() {
	m.Extend("default", driver.NewFile("", 120))
}
