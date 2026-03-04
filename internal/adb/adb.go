package adb

import (
	"os/exec"
	"strings"

	"autogetjs/pkg/logger"
)

// Client wraps ADB commands.
type Client struct{}

// NewClient creates a new ADB client.
func NewClient() *Client {
	return &Client{}
}

// Connect connects to a device over network (e.g. Gemphonefarm cloud: 42.114.191.99:5555).
func (c *Client) Connect(addr string) error {
	logger.Info("Connecting to %s...", addr)
	out, err := exec.Command("adb", "connect", addr).CombinedOutput()
	logger.Info("%s", strings.TrimSpace(string(out)))
	return err
}

// Devices returns connected device IDs from adb devices.
func (c *Client) Devices() ([]string, error) {
	logger.Info("Running adb devices...")
	out, err := exec.Command("adb", "devices").CombinedOutput()
	if err != nil {
		return nil, err
	}
	// Parse "List of devices attached" then lines like "42.114.191.99:5555	device"
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var ids []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && (parts[1] == "device" || parts[1] == "unauthorized") {
			ids = append(ids, parts[0])
		}
	}
	return ids, nil
}

// Run runs an adb command for the given device.
func (c *Client) Run(deviceID, command string) (string, error) {
	logger.Info("ADB %s: %s", deviceID, command)
	out, err := exec.Command("adb", "-s", deviceID, "shell", command).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
