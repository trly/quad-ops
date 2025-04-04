package unit

// Image represents the configuration for an image in a Quadlet unit.
type Image struct {
	Image      string   `yaml:"image"`
	PodmanArgs []string `yaml:"podman_args"`

	// Systemd unit properties
	Name     string
	UnitType string
}

// NewImage creates a new Image with the given name
func NewImage(name string) *Image {
	return &Image{
		Name:     name,
		UnitType: "image",
	}
}

// GetServiceName returns the full systemd service name
func (i *Image) GetServiceName() string {
	return i.Name + "-image.service"
}

// GetUnitType returns the type of the unit
func (i *Image) GetUnitType() string {
	return "image"
}

// GetUnitName returns the name of the unit
func (i *Image) GetUnitName() string {
	return i.Name
}

// GetStatus returns the current status of the unit
func (i *Image) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: i.Name, Type: "image"}
	return base.GetStatus()
}

// Start starts the unit
func (i *Image) Start() error {
	base := BaseSystemdUnit{Name: i.Name, Type: "image"}
	return base.Start()
}

// Stop stops the unit
func (i *Image) Stop() error {
	base := BaseSystemdUnit{Name: i.Name, Type: "image"}
	return base.Stop()
}

// Restart restarts the unit
func (i *Image) Restart() error {
	base := BaseSystemdUnit{Name: i.Name, Type: "image"}
	return base.Restart()
}

// Show displays the unit configuration and status
func (i *Image) Show() error {
	base := BaseSystemdUnit{Name: i.Name, Type: "image"}
	return base.Show()
}
