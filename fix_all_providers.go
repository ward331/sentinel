package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	providerDir := "internal/provider"
	
	// List all provider files
	files, err := ioutil.ReadDir(providerDir)
	if err != nil {
		panic(err)
	}
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".go") && file.Name() != "interface.go" && file.Name() != "common.go" {
			path := filepath.Join(providerDir, file.Name())
			fmt.Printf("Processing %s...\n", file.Name())
			
			content, err := ioutil.ReadFile(path)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
				continue
			}
			
			text := string(content)
			
			// Check if it has config field
			hasConfig := strings.Contains(text, "config *Config") || strings.Contains(text, "config  *Config")
			
			// Fix Enabled() method
			if strings.Contains(text, "func (p *") && strings.Contains(text, ") Enabled() bool {") {
				if hasConfig {
					// Replace with proper config check
					text = strings.Replace(text, 
						`if p.config != nil {
		return p.config.Enabled
	}
	return true`, 
						`if p.config != nil {
		return p.config.Enabled
	}
	return true`, -1)
				} else {
					// Provider without config - always enabled
					text = strings.Replace(text, 
						`if p.config != nil {
		return p.config.Enabled
	}
	return true`, 
						`return true`, -1)
				}
			}
			
			// Fix Interval() method
			if strings.Contains(text, "func (p *") && strings.Contains(text, ") Interval() time.Duration {") {
				if hasConfig {
					// Keep config-based interval
					text = strings.Replace(text, 
						`if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval`, 
						`if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval`, -1)
				} else {
					// Check for interval field
					if strings.Contains(text, "interval time.Duration") {
						text = strings.Replace(text, 
							`if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval`, 
							`return p.interval`, -1)
					} else if strings.Contains(text, "pollInterval time.Duration") {
						text = strings.Replace(text, 
							`if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval`, 
							`return p.pollInterval`, -1)
					} else {
						// Use default
						text = strings.Replace(text, 
							`if p.config != nil && p.config.PollInterval > 0 {
		return p.config.PollInterval
	}
	return 5 * time.Minute // Default interval`, 
							`return 5 * time.Minute // Default interval`, -1)
					}
				}
			}
			
			// Write back
			err = ioutil.WriteFile(path, []byte(text), 0644)
			if err != nil {
				fmt.Printf("  Error writing: %v\n", err)
			} else {
				fmt.Printf("  ✓ Fixed\n")
			}
		}
	}
	
	fmt.Println("Done!")
}