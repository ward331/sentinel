package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ServiceInfo holds service installation information
type ServiceInfo struct {
	Name        string
	DisplayName string
	Description string
	Executable  string
	Args        []string
	User        string
	WorkingDir  string
}

// DefaultServiceInfo returns default service configuration
func DefaultServiceInfo() *ServiceInfo {
	exePath, _ := os.Executable()
	if exePath == "" {
		exePath = "sentinel"
	}
	
	return &ServiceInfo{
		Name:        "sentinel",
		DisplayName: "SENTINEL World Event Monitor",
		Description: "Real-time world event monitoring and alerting system",
		Executable:  exePath,
		Args:        []string{"--data-dir", getDefaultDataDir()},
		User:        getCurrentUser(),
		WorkingDir:  getWorkingDir(),
	}
}

// Install installs the service on the current platform
func Install(info *ServiceInfo) error {
	switch runtime.GOOS {
	case "windows":
		return installWindowsService(info)
	case "darwin":
		return installMacOSService(info)
	case "linux":
		return installLinuxService(info)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Uninstall removes the service from the current platform
func Uninstall(name string) error {
	switch runtime.GOOS {
	case "windows":
		return uninstallWindowsService(name)
	case "darwin":
		return uninstallMacOSService(name)
	case "linux":
		return uninstallLinuxService(name)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Status checks if the service is installed and running
func Status(name string) (installed, running bool, err error) {
	switch runtime.GOOS {
	case "windows":
		return statusWindowsService(name)
	case "darwin":
		return statusMacOSService(name)
	case "linux":
		return statusLinuxService(name)
	default:
		return false, false, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Windows service implementation
func installWindowsService(info *ServiceInfo) error {
	// Check if we have admin privileges
	if !isWindowsAdmin() {
		return fmt.Errorf("administrator privileges required to install Windows service")
	}
	
	// Create service using sc.exe
	args := []string{
		"create",
		info.Name,
		"binPath=",
		fmt.Sprintf("\"%s\"", info.Executable),
	}
	
	// Add arguments
	for _, arg := range info.Args {
		args = append(args, arg)
	}
	
	// Add display name
	args = append(args, "DisplayName=", fmt.Sprintf("\"%s\"", info.DisplayName))
	
	// Set start type to auto
	args = append(args, "start=", "auto")
	
	cmd := exec.Command("sc.exe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create service: %v\n%s", err, output)
	}
	
	// Set description
	descCmd := exec.Command("sc.exe", "description", info.Name, info.Description)
	descOutput, descErr := descCmd.CombinedOutput()
	if descErr != nil {
		// Try to delete the service if description failed
		Uninstall(info.Name)
		return fmt.Errorf("failed to set service description: %v\n%s", descErr, descOutput)
	}
	
	fmt.Printf("Windows service '%s' installed successfully\n", info.Name)
	return nil
}

func uninstallWindowsService(name string) error {
	if !isWindowsAdmin() {
		return fmt.Errorf("administrator privileges required to uninstall Windows service")
	}
	
	// Stop the service first
	stopCmd := exec.Command("sc.exe", "stop", name)
	stopCmd.Run() // Ignore error, service might not be running
	
	// Delete the service
	cmd := exec.Command("sc.exe", "delete", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete service: %v\n%s", err, output)
	}
	
	fmt.Printf("Windows service '%s' uninstalled successfully\n", name)
	return nil
}

func statusWindowsService(name string) (installed, running bool, err error) {
	cmd := exec.Command("sc.exe", "query", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service not found
		return false, false, nil
	}
	
	outputStr := string(output)
	installed = true
	
	// Check if service is running
	if strings.Contains(outputStr, "RUNNING") {
		running = true
	}
	
	return installed, running, nil
}

func isWindowsAdmin() bool {
	// Simple check - in production would use proper Windows API
	// For now, assume not admin in this environment
	return false
}

// macOS service implementation (launchd)
func installMacOSService(info *ServiceInfo) error {
	// Create plist file
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>`, info.Name, info.Executable)
	
	// Add arguments
	for _, arg := range info.Args {
		plistContent += fmt.Sprintf("\n\t\t<string>%s</string>", arg)
	}
	
	plistContent += fmt.Sprintf(`
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/%s.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/%s.err</string>
	<key>WorkingDirectory</key>
	<string>%s</string>
</dict>
</plist>`, info.Name, info.Name, info.WorkingDir)
	
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", info.Name)
	
	// Check if we have permission to write to /Library/LaunchDaemons
	if !canWriteToLaunchDaemons() {
		return fmt.Errorf("root privileges required to install macOS service")
	}
	
	// Write plist file
	err := os.WriteFile(plistPath, []byte(plistContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write plist file: %v", err)
	}
	
	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up plist file
		os.Remove(plistPath)
		return fmt.Errorf("failed to load service: %v\n%s", err, output)
	}
	
	fmt.Printf("macOS service '%s' installed successfully\n", info.Name)
	return nil
}

func uninstallMacOSService(name string) error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", name)
	
	// Unload the service first
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	unloadCmd.Run() // Ignore error, service might not be loaded
	
	// Remove plist file
	err := os.Remove(plistPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %v", err)
	}
	
	fmt.Printf("macOS service '%s' uninstalled successfully\n", name)
	return nil
}

func statusMacOSService(name string) (installed, running bool, err error) {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", name)
	
	// Check if plist file exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return false, false, nil
	}
	installed = true
	
	// Check if service is running
	cmd := exec.Command("launchctl", "list", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service not loaded
		return installed, false, nil
	}
	
	outputStr := string(output)
	if strings.Contains(outputStr, name) && !strings.Contains(outputStr, "Could not find service") {
		running = true
	}
	
	return installed, running, nil
}

func canWriteToLaunchDaemons() bool {
	// Check if we can write to /Library/LaunchDaemons
	// For now, assume not root in this environment
	return false
}

// Linux service implementation (systemd)
func installLinuxService(info *ServiceInfo) error {
	// Create systemd service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=%s %s
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`, info.Description, info.User, info.WorkingDir, info.Executable, strings.Join(info.Args, " "))
	
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", info.Name)
	
	// Check if we have permission to write to /etc/systemd/system
	if !canWriteToSystemd() {
		return fmt.Errorf("root privileges required to install Linux service")
	}
	
	// Write service file
	err := os.WriteFile(servicePath, []byte(serviceContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}
	
	// Reload systemd
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	reloadOutput, reloadErr := reloadCmd.CombinedOutput()
	if reloadErr != nil {
		// Clean up service file
		os.Remove(servicePath)
		return fmt.Errorf("failed to reload systemd: %v\n%s", reloadErr, reloadOutput)
	}
	
	// Enable the service
	enableCmd := exec.Command("systemctl", "enable", info.Name)
	enableOutput, enableErr := enableCmd.CombinedOutput()
	if enableErr != nil {
		// Clean up service file
		os.Remove(servicePath)
		return fmt.Errorf("failed to enable service: %v\n%s", enableErr, enableOutput)
	}
	
	fmt.Printf("Linux service '%s' installed successfully\n", info.Name)
	return nil
}

func uninstallLinuxService(name string) error {
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", name)
	
	// Disable the service first
	disableCmd := exec.Command("systemctl", "disable", name)
	disableCmd.Run() // Ignore error, service might not be enabled
	
	// Stop the service
	stopCmd := exec.Command("systemctl", "stop", name)
	stopCmd.Run() // Ignore error, service might not be running
	
	// Remove service file
	err := os.Remove(servicePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %v", err)
	}
	
	// Reload systemd
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	reloadCmd.Run() // Ignore error
	
	fmt.Printf("Linux service '%s' uninstalled successfully\n", name)
	return nil
}

func statusLinuxService(name string) (installed, running bool, err error) {
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", name)
	
	// Check if service file exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return false, false, nil
	}
	installed = true
	
	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service not active
		return installed, false, nil
	}
	
	outputStr := strings.TrimSpace(string(output))
	running = (outputStr == "active")
	
	return installed, running, nil
}

func canWriteToSystemd() bool {
	// Check if we can write to /etc/systemd/system
	// For now, assume not root in this environment
	return false
}

// Helper functions
func getDefaultDataDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "SENTINEL", "data")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "SENTINEL", "data")
	default: // linux and others
		return filepath.Join(os.Getenv("HOME"), ".local", "share", "sentinel")
	}
}

func getCurrentUser() string {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	if user == "" {
		user = "root"
	}
	return user
}

func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return wd
}