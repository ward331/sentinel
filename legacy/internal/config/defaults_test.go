package config

import (
	"testing"
)

func TestDefaultConfig_Port(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestDefaultConfig_Version(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", cfg.Version)
	}
}

func TestDefaultConfig_SensibleValues(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LogLevel != "info" {
		t.Errorf("expected log level 'info', got %s", cfg.LogLevel)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.UI.DataRetentionDays != 30 {
		t.Errorf("expected data retention 30 days, got %d", cfg.UI.DataRetentionDays)
	}
	if cfg.Telegram.Enabled {
		t.Error("telegram should be disabled by default")
	}
	if cfg.Email.Enabled {
		t.Error("email should be disabled by default")
	}
	if !cfg.AutoOpenBrowser {
		t.Error("auto open browser should be true by default")
	}
}

func TestDefaultConfig_Tier0ProvidersEnabled(t *testing.T) {
	cfg := DefaultConfig()

	tier0 := []struct {
		name    string
		enabled bool
	}{
		{"USGS", cfg.Providers.USGS.Enabled},
		{"GDACS", cfg.Providers.GDACS.Enabled},
		{"OpenSky", cfg.Providers.OpenSky.Enabled},
		{"NOAA_CAP", cfg.Providers.NOAACAP.Enabled},
		{"OpenMeteo", cfg.Providers.OpenMeteo.Enabled},
		{"GDELT", cfg.Providers.GDELT.Enabled},
		{"Celestrak", cfg.Providers.Celestrak.Enabled},
		{"SWPC", cfg.Providers.SWPC.Enabled},
		{"WHO", cfg.Providers.WHO.Enabled},
		{"AirplanesLive", cfg.Providers.AirplanesLive.Enabled},
		{"NASAFIRMS", cfg.Providers.NASAFIRMS.Enabled},
		{"ReliefWeb", cfg.Providers.ReliefWeb.Enabled},
	}

	for _, p := range tier0 {
		if !p.enabled {
			t.Errorf("expected tier0 provider %s to be enabled by default", p.name)
		}
	}
}

func TestDefaultV3Config(t *testing.T) {
	cfg := DefaultV3Config()

	if cfg.Version != "3.0.0" {
		t.Errorf("expected V3 version 3.0.0, got %s", cfg.Version)
	}
	if !cfg.SignalBoard.Enabled {
		t.Error("expected signal board enabled in V3")
	}
	if !cfg.EntityTracking.Enabled {
		t.Error("expected entity tracking enabled in V3")
	}
	if cfg.EntityTracking.DeadReckoningMins != 30 {
		t.Errorf("expected dead reckoning 30 mins, got %d", cfg.EntityTracking.DeadReckoningMins)
	}
	if cfg.Location.RadiusKm != 100.0 {
		t.Errorf("expected proximity radius 100km, got %f", cfg.Location.RadiusKm)
	}
}

func TestDefaultConfig_GetDBPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataDir = "/tmp/sentinel-test"
	path := cfg.GetDBPath()
	if path != "/tmp/sentinel-test/sentinel.db" {
		t.Errorf("expected /tmp/sentinel-test/sentinel.db, got %s", path)
	}
}

func TestDefaultConfig_GetDBPath_Empty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataDir = ""
	path := cfg.GetDBPath()
	if path != "sentinel.db" {
		t.Errorf("expected fallback sentinel.db, got %s", path)
	}
}
