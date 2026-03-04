package device

import "autogetjs/pkg/logger"

// Device represents a connected device.
type Device struct {
	ID   string
	Name string
}

// Worker processes jobs for a single device.
type Worker struct {
	device *Device
}

// NewWorker creates a new worker for the given device.
func NewWorker(d *Device) *Worker {
	return &Worker{device: d}
}

// Run executes the worker loop.
func (w *Worker) Run(links []string) {
	logger.Info("Worker started for device %s", w.device.ID)
	for _, link := range links {
		w.process(link)
	}
}

func (w *Worker) process(link string) {
	// TODO: execute job for link on device
	logger.Info("Processing %s on device %s", link, w.device.ID)
}
