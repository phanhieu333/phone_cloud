package device

import (
	"autogetjs/internal/adb"
	"autogetjs/pkg/logger"
)

// Manager manages device lifecycle and state.
type Manager struct {
	devices []*Device
}

// NewManager creates a new device manager.
func NewManager() *Manager {
	return &Manager{
		devices: make([]*Device, 0),
	}
}

// GetDevices returns all connected devices. Pass cloud IPs (e.g. "42.114.191.99:5555")
// to connect via ADB before listing; leave nil to only list already-connected devices.
func GetDevices(connectIPs []string) ([]*Device, error) {
	logger.Info("Fetching devices...")
	client := adb.NewClient()
	for _, ip := range connectIPs {
		if err := client.Connect(ip); err != nil {
			logger.Error("Connect %s: %v", ip, err)
		}
	}
	ids, err := client.Devices()
	if err != nil {
		return nil, err
	}
	list := make([]*Device, 0, len(ids))
	for _, id := range ids {
		list = append(list, &Device{ID: id, Name: id})
	}
	return list, nil
}

// AddDevice adds a device to the manager.
func (m *Manager) AddDevice(d *Device) {
	m.devices = append(m.devices, d)
}

// Devices returns the list of managed devices.
func (m *Manager) Devices() []*Device {
	return m.devices
}
