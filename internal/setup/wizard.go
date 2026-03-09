package setup

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/openclaw/sentinel-backend/internal/config"
)

// Wizard handles the first-run setup process
type Wizard struct {
	config *config.Config
}

// NewWizard creates a new setup wizard
func NewWizard(cfg *config.Config) *Wizard {
	return &Wizard{
		config: cfg,
	}
}

// Run runs the interactive setup wizard
func (w *Wizard) Run() error {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║            SENTINEL Setup Wizard                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Welcome to SENTINEL! Let's configure your system.")
	fmt.Println()
	
	// Step 1: Data directory
	w.askDataDirectory()
	
	// Step 2: Server configuration
	w.askServerConfig()
	
	// Step 3: Cesium token
	w.askCesiumToken()
	
	// Step 4: Notification methods
	w.askNotificationMethods()
	
	// Step 5: Providers
	w.askProviders()
	
	// Step 6: Location
	w.askLocation()
	
	// Step 7: UI preferences
	w.askUIPreferences()
	
	// Save configuration
	return w.saveConfig()
}

// askDataDirectory asks for data directory
func (w *Wizard) askDataDirectory() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 1: Data Storage")
	fmt.Println("════════════════════════════════════════════════════════")
	
	defaultDir := config.GetDefaultDataDir()
	fmt.Printf("Where should SENTINEL store its data?\n")
	fmt.Printf("Default: %s\n", defaultDir)
	fmt.Print("Press Enter to use default, or enter custom path: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		w.config.DataDir = defaultDir
	} else {
		w.config.DataDir = input
	}
	
	fmt.Printf("Data will be stored in: %s\n", w.config.DataDir)
	fmt.Println()
}

// askServerConfig asks for server configuration
func (w *Wizard) askServerConfig() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 2: Server Configuration")
	fmt.Println("════════════════════════════════════════════════════════")
	
	// Port
	fmt.Print("Server port (default 8080): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input != "" {
		if port, err := strconv.Atoi(input); err == nil && port > 0 && port < 65536 {
			w.config.Server.Port = port
		} else {
			fmt.Printf("Invalid port, using default 8080\n")
			w.config.Server.Port = 8080
		}
	} else {
		w.config.Server.Port = 8080
	}
	
	// Host
	fmt.Print("Bind host (default 0.0.0.0 for all interfaces): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input != "" {
		w.config.Server.Host = input
	} else {
		w.config.Server.Host = "0.0.0.0"
	}
	
	// Auto-open browser
	fmt.Print("Auto-open browser on startup? (Y/n): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	w.config.AutoOpenBrowser = (input == "" || input == "y" || input == "yes")
	
	fmt.Printf("Server will run on: %s:%d\n", w.config.Server.Host, w.config.Server.Port)
	fmt.Println()
}

// askCesiumToken asks for Cesium Ion token
func (w *Wizard) askCesiumToken() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 3: Cesium Ion Token")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("SENTINEL uses CesiumJS for 3D globe visualization.")
	fmt.Println("You need a Cesium Ion token for satellite imagery.")
	fmt.Println()
	fmt.Println("Get a free token at: https://cesium.com/ion/tokens")
	fmt.Println("(Required for Earth satellite imagery)")
	fmt.Println()
	
	fmt.Print("Enter your Cesium Ion token (or press Enter to skip): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input != "" {
		w.config.CesiumToken = input
		fmt.Println("✓ Cesium token saved")
	} else {
		fmt.Println("⚠  No Cesium token provided. Globe will show basic colors.")
	}
	
	fmt.Println()
}

// askNotificationMethods asks about notification methods
func (w *Wizard) askNotificationMethods() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 4: Notification Methods")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("How would you like to receive alerts?")
	fmt.Println()
	
	reader := bufio.NewReader(os.Stdin)
	
	// Telegram
	fmt.Print("Enable Telegram notifications? (y/N): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "y" || input == "yes" {
		w.config.Telegram.Enabled = true
		fmt.Print("Telegram bot token: ")
		token, _ := reader.ReadString('\n')
		w.config.Telegram.BotToken = strings.TrimSpace(token)
		
		fmt.Print("Telegram chat ID: ")
		chatID, _ := reader.ReadString('\n')
		w.config.Telegram.ChatID = strings.TrimSpace(chatID)
		
		fmt.Println("✓ Telegram configured")
	}
	
	// Other methods would be similar
	fmt.Println("Other notification methods can be configured later in settings.")
	fmt.Println()
}

// askProviders asks which data providers to enable
func (w *Wizard) askProviders() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 5: Data Providers")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Which data sources would you like to monitor?")
	fmt.Println()
	
	reader := bufio.NewReader(os.Stdin)
	
	// USGS (Earthquakes)
	fmt.Print("Enable USGS earthquake data? (Y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	w.config.Providers.USGS.Enabled = (input == "" || input == "y" || input == "yes")
	
	// GDACS (Disasters)
	fmt.Print("Enable GDACS disaster alerts? (Y/n): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	w.config.Providers.GDACS.Enabled = (input == "" || input == "y" || input == "yes")
	
	// OpenSky (Flights)
	fmt.Print("Enable OpenSky flight data? (Y/n): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	w.config.Providers.OpenSky.Enabled = (input == "" || input == "y" || input == "yes")
	
	fmt.Println("✓ Providers configured")
	fmt.Println("Other providers can be enabled later in settings.")
	fmt.Println()
}

// askLocation asks for user location
func (w *Wizard) askLocation() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 6: Your Location")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Your location helps with local alerts and ISS passes.")
	fmt.Println()
	
	fmt.Print("Would you like to set your location now? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "y" || input == "yes" {
		// Latitude
		fmt.Print("Latitude (e.g., 40.7128 for New York): ")
		latStr, _ := reader.ReadString('\n')
		latStr = strings.TrimSpace(latStr)
		
		// Longitude
		fmt.Print("Longitude (e.g., -74.0060 for New York): ")
		lonStr, _ := reader.ReadString('\n')
		lonStr = strings.TrimSpace(lonStr)
		
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			w.config.Location.Lat = lat
		}
		
		if lon, err := strconv.ParseFloat(lonStr, 64); err == nil {
			w.config.Location.Lon = lon
		}
		
		if w.config.Location.Lat != 0 || w.config.Location.Lon != 0 {
			w.config.Location.Set = true
			fmt.Printf("✓ Location set to: %.4f, %.4f\n", w.config.Location.Lat, w.config.Location.Lon)
		}
	} else {
		fmt.Println("⚠  Location not set. Can be configured later.")
	}
	
	fmt.Println()
}

// askUIPreferences asks for UI preferences
func (w *Wizard) askUIPreferences() {
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Step 7: UI Preferences")
	fmt.Println("════════════════════════════════════════════════════════")
	
	reader := bufio.NewReader(os.Stdin)
	
	// Default view
	fmt.Print("Default view (globe/map/list, default globe): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "map" || input == "list" {
		w.config.UI.DefaultView = input
	} else {
		w.config.UI.DefaultView = "globe"
	}
	
	// Sound
	fmt.Print("Enable sound alerts? (Y/n): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	w.config.UI.SoundEnabled = (input == "" || input == "y" || input == "yes")
	
	fmt.Println("✓ UI preferences saved")
	fmt.Println()
}

// saveConfig saves the configuration
func (w *Wizard) saveConfig() error {
	// Mark setup as complete
	w.config.SetupComplete = true
	
	// Save to default config path
	configPath := config.GetDefaultConfigPath()
	err := config.SaveConfig(w.config, configPath)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}
	
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Setup Complete!")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Start SENTINEL: ./sentinel")
	fmt.Println("2. Open dashboard: http://localhost:8080")
	fmt.Println("3. Configure additional settings in the web interface")
	fmt.Println()
	fmt.Println("Thank you for choosing SENTINEL!")
	
	return nil
}

// RunIfNeeded runs the setup wizard if configuration is not complete
func RunIfNeeded(cfg *config.Config) error {
	if cfg.SetupComplete {
		return nil
	}
	
	wizard := NewWizard(cfg)
	return wizard.Run()
}