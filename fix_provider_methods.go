package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	// Map of provider files to their correct struct names
	providers := map[string]string{
		"adsb_one.go":           "ADSBOneProvider",
		"airplanes_live.go":     "AirplanesLiveProvider",
		"celestrak.go":          "CelesTrakProvider",
		"financial_markets.go":  "FinancialMarketsProvider",
		"gdacs.go":              "GDACSProvider",
		"gdelt.go":              "GDELTProvider",
		"globalfishingwatch.go": "GlobalFishingWatchProvider",
		"globalforestwatch.go":  "GlobalForestWatchProvider",
		"iranconflict.go":       "IranConflictProvider",
		"liveuamap.go":          "LiveUAMapProvider",
		"nasa_firms.go":         "NASAFIRMSProvider",
		"noaa_cap.go":           "NOAACAPProvider",
		"noaa_nws.go":           "NOAANWSProvider",
		"openmeteo.go":          "OpenMeteoProvider",
		"opensanctions.go":      "OpenSanctionsProvider",
		"opensky_enhanced.go":   "OpenSkyEnhancedProvider",
		"piracy_imb.go":         "PiracyIMBProvider",
		"promed.go":             "ProMEDProvider",
		"reliefweb.go":          "ReliefWebProvider",
		"swpc.go":               "SWPCProvider",
		"tsunami.go":            "TsunamiProvider",
		"usgs.go":               "USGSProvider",
		"volcano.go":            "VolcanoProvider",
		"who.go":                "WHOProvider",
	}

	for filename, structName := range providers {
		path := filepath.Join("internal/provider", filename)
		fmt.Printf("Processing %s (%s)...\n", filename, structName)
		
		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("  Error reading: %v\n", err)
			continue
		}
		
		text := string(content)
		modified := false
		
		// Check and add Name() method
		if !strings.Contains(text, "func (p *"+structName+") Name() string {") {
			// Find Fetch method
			fetchIndex := strings.Index(text, "func (p *"+structName+") Fetch")
			if fetchIndex == -1 {
				// Try alternative pattern
				fetchIndex = strings.Index(text, "func (p *"+strings.ToLower(structName[:1])+structName[1:]+") Fetch")
			}
			
			if fetchIndex > 0 {
				// Insert Name method before Fetch
				nameMethod := "\n// Name returns the provider identifier\n" +
					"func (p *" + structName + ") Name() string {\n" +
					"\treturn \"" + strings.ToLower(strings.ReplaceAll(structName, "Provider", "")) + "\"\n" +
					"}\n\n"
				
				text = text[:fetchIndex] + nameMethod + text[fetchIndex:]
				modified = true
				fmt.Printf("  Added Name() method\n")
			}
		}
		
		// Check and add Enabled() method
		if !strings.Contains(text, "func (p *"+structName+") Enabled() bool {") {
			// Check if provider has config field
			hasConfig := strings.Contains(text, "config *Config") || strings.Contains(text, "config  *Config")
			
			fetchIndex := strings.Index(text, "func (p *"+structName+") Fetch")
			if fetchIndex == -1 {
				fetchIndex = strings.Index(text, "func (p *"+strings.ToLower(structName[:1])+structName[1:]+") Fetch")
			}
			
			if fetchIndex > 0 {
				var enabledMethod string
				if hasConfig {
					enabledMethod = "\n// Enabled returns whether the provider is enabled\n" +
						"func (p *" + structName + ") Enabled() bool {\n" +
						"\tif p.config != nil {\n" +
						"\t\treturn p.config.Enabled\n" +
						"\t}\n" +
						"\treturn true\n" +
						"}\n\n"
				} else {
					enabledMethod = "\n// Enabled returns whether the provider is enabled\n" +
						"func (p *" + structName + ") Enabled() bool {\n" +
						"\treturn true\n" +
						"}\n\n"
				}
				
				text = text[:fetchIndex] + enabledMethod + text[fetchIndex:]
				modified = true
				fmt.Printf("  Added Enabled() method\n")
			}
		}
		
		// Check and add Interval() method
		if !strings.Contains(text, "func (p *"+structName+") Interval() time.Duration {") {
			// Check if provider has config field or interval field
			hasConfig := strings.Contains(text, "config *Config") || strings.Contains(text, "config  *Config")
			hasIntervalField := strings.Contains(text, "interval time.Duration")
			
			fetchIndex := strings.Index(text, "func (p *"+structName+") Fetch")
			if fetchIndex == -1 {
				fetchIndex = strings.Index(text, "func (p *"+strings.ToLower(structName[:1])+structName[1:]+") Fetch")
			}
			
			if fetchIndex > 0 {
				var intervalMethod string
				if hasConfig {
					intervalMethod = "\n// Interval returns the polling interval\n" +
						"func (p *" + structName + ") Interval() time.Duration {\n" +
						"\tif p.config != nil && p.config.PollInterval > 0 {\n" +
						"\t\treturn p.config.PollInterval\n" +
						"\t}\n" +
						"\treturn 5 * time.Minute // Default interval\n" +
						"}\n\n"
				} else if hasIntervalField {
					intervalMethod = "\n// Interval returns the polling interval\n" +
						"func (p *" + structName + ") Interval() time.Duration {\n" +
						"\treturn p.interval\n" +
						"}\n\n"
				} else {
					intervalMethod = "\n// Interval returns the polling interval\n" +
						"func (p *" + structName + ") Interval() time.Duration {\n" +
						"\treturn 5 * time.Minute // Default interval\n" +
						"}\n\n"
				}
				
				text = text[:fetchIndex] + intervalMethod + text[fetchIndex:]
				modified = true
				fmt.Printf("  Added Interval() method\n")
			}
		}
		
		if modified {
			err = ioutil.WriteFile(path, []byte(text), 0644)
			if err != nil {
				fmt.Printf("  Error writing: %v\n", err)
			}
		}
	}
	
	fmt.Println("Done!")
}