package tray

import (
	"fmt"
	"runtime"

	"github.com/getlantern/systray"
)

// Tray represents the system tray application
type Tray struct {
	onOpenDashboard func()
	onOpenSettings  func()
	onQuit          func()
	config          *Config
}

// Config holds tray configuration
type Config struct {
	AppName      string
	Version      string
	DashboardURL string
	IconPath     string
}

// New creates a new system tray
func New(config *Config, onOpenDashboard, onOpenSettings, onQuit func()) *Tray {
	return &Tray{
		onOpenDashboard: onOpenDashboard,
		onOpenSettings:  onOpenSettings,
		onQuit:          onQuit,
		config:          config,
	}
}

// Start starts the system tray
func (t *Tray) Start() {
	// Run on the main thread (required by systray)
	systray.Run(t.onReady, t.onExit)
}

// Stop stops the system tray
func (t *Tray) Stop() {
	systray.Quit()
}

// onReady is called when systray is ready
func (t *Tray) onReady() {
	// Set icon
	if t.config.IconPath != "" {
		systray.SetIcon(loadIcon(t.config.IconPath))
	} else {
		// Use default icon (embedded or generated)
		systray.SetIcon(getDefaultIcon())
	}
	
	// Set tooltip
	systray.SetTooltip(fmt.Sprintf("%s v%s", t.config.AppName, t.config.Version))
	
	// Create menu items
	mDashboard := systray.AddMenuItem("Open Dashboard", "Open web dashboard")
	mSettings := systray.AddMenuItem("Settings", "Open settings")
	systray.AddSeparator()
	
	// Platform-specific items
	if runtime.GOOS == "darwin" {
		// macOS has a "About" menu item convention
		mAbout := systray.AddMenuItem(fmt.Sprintf("About %s", t.config.AppName), "About this application")
		go func() {
			for {
				select {
				case <-mAbout.ClickedCh:
					showAboutDialog(t.config)
				}
			}
		}()
		systray.AddSeparator()
	}
	
	mQuit := systray.AddMenuItem("Quit", "Quit application")
	
	// Handle menu events
	go t.handleMenuEvents(mDashboard, mSettings, mQuit)
	
	// Show notification on startup
	t.showStartupNotification()
}

// onExit is called when systray exits
func (t *Tray) onExit() {
	// Cleanup if needed
}

// handleMenuEvents handles menu item clicks
func (t *Tray) handleMenuEvents(mDashboard, mSettings, mQuit *systray.MenuItem) {
	for {
		select {
		case <-mDashboard.ClickedCh:
			if t.onOpenDashboard != nil {
				t.onOpenDashboard()
			}
		case <-mSettings.ClickedCh:
			if t.onOpenSettings != nil {
				t.onOpenSettings()
			}
		case <-mQuit.ClickedCh:
			if t.onQuit != nil {
				t.onQuit()
			}
			systray.Quit()
		}
	}
}

// showStartupNotification shows a notification when the app starts
func (t *Tray) showStartupNotification() {
	// Platform-specific notifications
	switch runtime.GOOS {
	case "windows":
		// Windows toast notification
		showWindowsNotification(fmt.Sprintf("%s is running", t.config.AppName), 
			"Click the tray icon to open dashboard")
	case "darwin":
		// macOS notification
		showMacOSNotification(fmt.Sprintf("%s is running", t.config.AppName), 
			"Running in background")
	case "linux":
		// Linux notification (using libnotify)
		showLinuxNotification(fmt.Sprintf("%s is running", t.config.AppName), 
			"Running in background")
	}
}

// loadIcon loads icon from file
func loadIcon(path string) []byte {
	// In a real implementation, this would load from file
	// For now, return default icon
	return getDefaultIcon()
}

// getDefaultIcon returns a default icon
func getDefaultIcon() []byte {
	// Simple generated icon (16x16 PNG)
	// In production, this would be embedded or loaded from assets
	return generateDefaultIcon()
}

// generateDefaultIcon generates a simple default icon
func generateDefaultIcon() []byte {
	// This is a minimal 16x16 black/white icon as PNG
	// In a real implementation, use proper PNG generation or embedded asset
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG header
		// ... rest of PNG data would go here
	}
}

// Platform-specific notification functions
func showWindowsNotification(title, message string) {
	// Would use Windows toast notifications API
	// For now, just log
	fmt.Printf("Windows notification: %s - %s\n", title, message)
}

func showMacOSNotification(title, message string) {
	// Would use macOS notification API
	// For now, just log
	fmt.Printf("macOS notification: %s - %s\n", title, message)
}

func showLinuxNotification(title, message string) {
	// Would use libnotify or similar
	// For now, just log
	fmt.Printf("Linux notification: %s - %s\n", title, message)
}

// showAboutDialog shows about dialog (macOS specific)
func showAboutDialog(config *Config) {
	// Would show native about dialog on macOS
	// For now, just log
	fmt.Printf("About %s v%s\n", config.AppName, config.Version)
}