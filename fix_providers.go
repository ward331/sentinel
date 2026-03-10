package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	providerDir := "internal/provider"
	
	// Map of provider files to their struct names
	providers := map[string]string{
		"celestrak.go":           "CelesTrakProvider",
		"gdelt.go":               "GDELTProvider",
		"globalfishingwatch.go":  "GlobalFishingWatchProvider",
		"globalforestwatch.go":   "GlobalForestWatchProvider",
		"iranconflict.go":        "IranConflictProvider",
		"liveuamap.go":           "LiveUAMapProvider",
		"noaa_cap.go":            "NOAACAPProvider",
		"noaa_nws.go":            "NOAANWSProvider",
		"opensanctions.go":       "OpenSanctionsProvider",
		"piracy_imb.go":          "PiracyIMBProvider",
		"promed.go":              "ProMEDProvider",
		"swpc.go":                "SWPCProvider",
	}
	
	for filename, structName := range providers {
		path := filepath.Join(providerDir, filename)
		fmt.Printf("Processing %s (%s)...\n", filename, structName)
		
		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("  Error reading file: %v\n", err)
			continue
		}
		
		lines := strings.Split(string(content), "\n")
		var newLines []string
		foundFetch := false
		
		for i, line := range lines {
			newLines = append(newLines, line)
			
			// Look for the Fetch method
			if strings.Contains(line, "func (p *"+structName+") Fetch") && !foundFetch {
				foundFetch = true
				
				// Insert Enabled() and Interval() methods before Fetch
				methods := []string{
					"",
					"// Enabled returns whether the provider is enabled",
					"func (p *" + structName + ") Enabled() bool {",
					"\tif p.config != nil {",
					"\t\treturn p.config.Enabled",
					"\t}",
					"\treturn true",
					"}",
					"",
					"// Interval returns the polling interval",
					"func (p *" + structName + ") Interval() time.Duration {",
					"\tif p.config != nil && p.config.PollInterval > 0 {",
					"\t\treturn p.config.PollInterval",
					"\t}",
					"\treturn 5 * time.Minute // Default interval",
					"}",
					"",
				}
				
				// Insert methods in reverse order to maintain correct position
				for j := len(methods) - 1; j >= 0; j-- {
					newLines = append(newLines[:i], append([]string{methods[j]}, newLines[i:]...)...)
				}
			}
		}
		
		if foundFetch {
			// Write the file back
			output := strings.Join(newLines, "\n")
			err = ioutil.WriteFile(path, []byte(output), 0644)
			if err != nil {
				fmt.Printf("  Error writing file: %v\n", err)
			} else {
				fmt.Printf("  ✓ Added methods to %s\n", structName)
			}
		} else {
			fmt.Printf("  ✗ Could not find Fetch method for %s\n", structName)
		}
	}
	
	fmt.Println("Done!")
}