package session

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rat/session/contract"
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
	key        string
	lifetime   int
	gcInterval int
	drivers    map[string]contract.Driver
}

// NewManager creates a new session manager.
func NewManager(option *ManagerOptions) *Manager {
	manager := &Manager{
		key:        option.Key,
		lifetime:   option.Lifetime,
		gcInterval: option.GcInterval,
		drivers:    make(map[string]contract.Driver),
	}
	manager.createDefaultDriver()
	return manager
}

func (m *Manager) BuildSession(name, driver string, sessionID ...string) (contract.Session, error) {
	handler, err := m.driver(driver)
	if err != nil {
		return nil, err
	}

	return NewSession(name, m.key, int64(m.lifetime), handler, sessionID...)
}

func (m *Manager) Extend(driver string, handler contract.Driver) contract.Manager {
	m.drivers[driver] = handler
	m.startGcTimer(m.drivers[driver])
	return m
}

func (m *Manager) driver(name ...string) (contract.Driver, error) {
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

func (m *Manager) startGcTimer(driver contract.Driver) {
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
