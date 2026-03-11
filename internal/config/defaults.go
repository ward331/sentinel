package config

// DefaultV3Config returns sensible defaults for V3
// All tier0 providers enabled, port 8080, signal board on
func DefaultV3Config() *Config {
	cfg := DefaultConfig()
	cfg.Version = "3.0.0"

	// V3 additions — proximity alerts
	cfg.Location.RadiusKm = 100.0

	// Signal board on by default
	cfg.SignalBoard = SignalBoardConfig{Enabled: true}

	// Entity tracking defaults
	cfg.EntityTracking = EntityTrackingConfig{
		Enabled:           true,
		DeadReckoningMins: 30,
	}

	return cfg
}
