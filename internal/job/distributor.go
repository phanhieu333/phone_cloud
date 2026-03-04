package job

import (
	"autogetjs/internal/device"
	"autogetjs/pkg/logger"
)

// Distributor distributes jobs across devices.
type Distributor struct {
	devices []*device.Device
	links   []string
}

// NewDistributor creates a new job distributor.
func NewDistributor(devices []*device.Device, links []string) *Distributor {
	return &Distributor{
		devices: devices,
		links:   links,
	}
}

// Run distributes links across devices and runs workers.
func Run(devices []*device.Device, links []string) {
	d := NewDistributor(devices, links)
	d.Distribute()
}

// Distribute assigns links to devices and starts workers.
func (d *Distributor) Distribute() {
	logger.Info("Distributing %d links across %d devices", len(d.links), len(d.devices))
	if len(d.devices) == 0 {
		logger.Info("No devices available")
		return
	}
	for i, dev := range d.devices {
		linksForDevice := d.linksForDevice(i)
		w := device.NewWorker(dev)
		w.Run(linksForDevice)
	}
}

func (d *Distributor) linksForDevice(deviceIndex int) []string {
	count := len(d.devices)
	var result []string
	for i := deviceIndex; i < len(d.links); i += count {
		result = append(result, d.links[i])
	}
	return result
}
