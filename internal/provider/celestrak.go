package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// CelesTrakProvider fetches satellite tracking data from CelesTrak
type CelesTrakProvider struct {
	client *http.Client
	config *Config
}




// NewCelesTrakProvider creates a new CelesTrakProvider
func NewCelesTrakProvider(config *Config) *CelesTrakProvider {
	return &CelesTrakProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// Fetch retrieves satellite tracking data from CelesTrak
func (p *CelesTrakProvider) Fetch(ctx context.Context) ([]*model.Event, error) {
	var allEvents []*model.Event
	
	// Fetch from all CelesTrak sub-providers
	subProviders := []struct {
		name string
		url  string
	}{
		{"celestrak_active", "https://celestrak.org/NORAD/elements/gp.php?GROUP=active&FORMAT=tle"},
		{"celestrak_stations", "https://celestrak.org/NORAD/elements/gp.php?GROUP=stations&FORMAT=tle"},
		{"celestrak_starlink", "https://celestrak.org/NORAD/elements/gp.php?GROUP=starlink&FORMAT=tle"},
		{"celestrak_gps", "https://celestrak.org/NORAD/elements/gp.php?GROUP=gps-ops&FORMAT=tle"},
	}
	
	for _, sp := range subProviders {
		events, err := p.fetchSatelliteGroup(ctx, sp.name, sp.url)
		if err != nil {
			// Log error but continue with other groups
			fmt.Printf("Warning: Failed to fetch %s: %v\n", sp.name, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}
	
	return allEvents, nil
}

// fetchSatelliteGroup fetches TLE data for a specific satellite group
func (p *CelesTrakProvider) fetchSatelliteGroup(ctx context.Context, groupName, url string) ([]*model.Event, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", groupName, err)
	}
	
	req.Header.Set("User-Agent", "SENTINEL/2.0 (https://github.com/ward331/sentinel)")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s data: %w", groupName, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s returned status %d: %s", groupName, resp.StatusCode, string(body))
	}
	
	// Read TLE data
	tleData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s response: %w", groupName, err)
	}
	
	return p.parseTLEData(groupName, string(tleData))
}

// parseTLEData parses Two-Line Element (TLE) data
func (p *CelesTrakProvider) parseTLEData(groupName, tleData string) ([]*model.Event, error) {
	lines := strings.Split(tleData, "\n")
	var events []*model.Event
	
	// TLE format: name line, line 1, line 2 (repeat)
	for i := 0; i+2 < len(lines); i += 3 {
		nameLine := strings.TrimSpace(lines[i])
		line1 := strings.TrimSpace(lines[i+1])
		line2 := strings.TrimSpace(lines[i+2])
		
		// Skip if any line is empty
		if nameLine == "" || line1 == "" || line2 == "" {
			continue
		}
		
		// Parse satellite information from TLE
		satellite, err := p.parseSatelliteFromTLE(nameLine, line1, line2)
		if err != nil {
			fmt.Printf("Warning: Failed to parse TLE for %s: %v\n", nameLine, err)
			continue
		}
		
		// Create event from satellite data
		event := &model.Event{
			Title:       p.generateTitle(satellite),
			Description: p.generateDescription(satellite),
			Source:      groupName,
			SourceID:    satellite.NORADID,
			OccurredAt:  time.Now().UTC(),
			Location:    p.calculateOrbitLocation(satellite),
			Precision:   model.PrecisionApproximate,
			Magnitude:   p.calculateMagnitude(satellite),
			Category:    "satellite",
			Severity:    model.SeverityLow,
			Metadata:    p.generateMetadata(satellite),
			Badges:      p.generateBadges(satellite),
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// Satellite represents parsed TLE data
type Satellite struct {
	Name           string
	NORADID        string
	Classification string
	LaunchYear     string
	LaunchNumber   string
	LaunchPiece    string
	EpochYear      string
	EpochDay       string
	FirstDeriv     string
	SecondDeriv    string
	BStar          string
	EphemerisType  string
	ElementNumber  string
	Inclination    float64
	RAAN           float64
	Eccentricity   float64
	ArgPerigee     float64
	MeanAnomaly    float64
	MeanMotion     float64
	RevolutionNum  string
}

// parseSatelliteFromTLE parses TLE lines into Satellite struct
func (p *CelesTrakProvider) parseSatelliteFromTLE(nameLine, line1, line2 string) (*Satellite, error) {
	sat := &Satellite{
		Name: strings.TrimSpace(nameLine),
	}
	
	// Parse line 1
	if len(line1) >= 69 {
		sat.NORADID = strings.TrimSpace(line1[2:7])
		sat.Classification = strings.TrimSpace(line1[7:8])
		sat.LaunchYear = strings.TrimSpace(line1[9:11])
		sat.LaunchNumber = strings.TrimSpace(line1[11:14])
		sat.LaunchPiece = strings.TrimSpace(line1[14:17])
		sat.EpochYear = strings.TrimSpace(line1[18:20])
		sat.EpochDay = strings.TrimSpace(line1[20:32])
		sat.FirstDeriv = strings.TrimSpace(line1[33:43])
		sat.SecondDeriv = strings.TrimSpace(line1[44:52])
		sat.BStar = strings.TrimSpace(line1[53:61])
		sat.EphemerisType = strings.TrimSpace(line1[62:63])
		sat.ElementNumber = strings.TrimSpace(line1[64:68])
	}
	
	// Parse line 2
	if len(line2) >= 69 {
		// Parse inclination
		if incStr := strings.TrimSpace(line2[8:16]); incStr != "" {
			if inc, err := strconv.ParseFloat(incStr, 64); err == nil {
				sat.Inclination = inc
			}
		}
		
		// Parse RAAN (Right Ascension of Ascending Node)
		if raanStr := strings.TrimSpace(line2[17:25]); raanStr != "" {
			if raan, err := strconv.ParseFloat(raanStr, 64); err == nil {
				sat.RAAN = raan
			}
		}
		
		// Parse eccentricity
		if eccStr := strings.TrimSpace(line2[26:33]); eccStr != "" {
			if ecc, err := strconv.ParseFloat("0."+eccStr, 64); err == nil {
				sat.Eccentricity = ecc
			}
		}
		
		// Parse argument of perigee
		if argPerigeeStr := strings.TrimSpace(line2[34:42]); argPerigeeStr != "" {
			if argPerigee, err := strconv.ParseFloat(argPerigeeStr, 64); err == nil {
				sat.ArgPerigee = argPerigee
			}
		}
		
		// Parse mean anomaly
		if meanAnomalyStr := strings.TrimSpace(line2[43:51]); meanAnomalyStr != "" {
			if meanAnomaly, err := strconv.ParseFloat(meanAnomalyStr, 64); err == nil {
				sat.MeanAnomaly = meanAnomaly
			}
		}
		
		// Parse mean motion
		if meanMotionStr := strings.TrimSpace(line2[52:63]); meanMotionStr != "" {
			if meanMotion, err := strconv.ParseFloat(meanMotionStr, 64); err == nil {
				sat.MeanMotion = meanMotion
			}
		}
		
		// Parse revolution number
		if len(line2) >= 68 {
			sat.RevolutionNum = strings.TrimSpace(line2[63:68])
		}
	}
	
	return sat, nil
}

// generateTitle creates a title for the satellite event
func (p *CelesTrakProvider) generateTitle(sat *Satellite) string {
	return fmt.Sprintf("🛰️ %s (NORAD %s)", sat.Name, sat.NORADID)
}

// generateDescription creates a description for the satellite event
func (p *CelesTrakProvider) generateDescription(sat *Satellite) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Satellite: %s\n", sat.Name))
	builder.WriteString(fmt.Sprintf("NORAD ID: %s\n", sat.NORADID))
	builder.WriteString(fmt.Sprintf("Classification: %s\n", sat.Classification))
	
	if sat.LaunchYear != "" {
		builder.WriteString(fmt.Sprintf("Launch Year: %s\n", sat.LaunchYear))
	}
	
	builder.WriteString(fmt.Sprintf("Orbital Parameters:\n"))
	builder.WriteString(fmt.Sprintf("  • Inclination: %.2f°\n", sat.Inclination))
	builder.WriteString(fmt.Sprintf("  • RAAN: %.2f°\n", sat.RAAN))
	builder.WriteString(fmt.Sprintf("  • Eccentricity: %.6f\n", sat.Eccentricity))
	builder.WriteString(fmt.Sprintf("  • Mean Motion: %.4f rev/day\n", sat.MeanMotion))
	
	if sat.RevolutionNum != "" {
		builder.WriteString(fmt.Sprintf("  • Revolution #: %s\n", sat.RevolutionNum))
	}
	
	builder.WriteString("\nSource: CelesTrak NORAD Two-Line Element (TLE) Data")
	builder.WriteString("\nUpdated: " + time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	
	return builder.String()
}

// calculateOrbitLocation calculates approximate ground track location
func (p *CelesTrakProvider) calculateOrbitLocation(sat *Satellite) model.GeoJSON {
	// Simplified: Use mean anomaly to estimate position
	// In reality, would need SGP4 propagation
	// For now, return a point on the equator at RAAN longitude
	
	lon := sat.RAAN
	if lon > 180 {
		lon = lon - 360
	}
	
	return model.GeoJSON{
		Type:        "Point",
		Coordinates: []float64{lon, 0.0}, // Equator at RAAN longitude
	}
}

// calculateMagnitude calculates magnitude based on satellite characteristics
func (p *CelesTrakProvider) calculateMagnitude(sat *Satellite) float64 {
	magnitude := 3.0 // Base for satellite tracking
	
	// Adjust based on orbit characteristics
	if sat.Inclination > 80 {
		magnitude += 1.0 // Polar orbit
	}
	if sat.MeanMotion > 15 {
		magnitude += 0.5 // Low Earth Orbit (fast)
	}
	if strings.Contains(strings.ToLower(sat.Name), "starlink") {
		magnitude += 0.5 // Starlink constellation
	}
	if strings.Contains(strings.ToLower(sat.Name), "gps") {
		magnitude += 1.0 // GPS navigation
	}
	if strings.Contains(strings.ToLower(sat.Name), "iss") || strings.Contains(strings.ToLower(sat.Name), "station") {
		magnitude += 1.5 // Space station
	}
	
	return magnitude
}

// generateMetadata creates metadata for the satellite event
func (p *CelesTrakProvider) generateMetadata(sat *Satellite) map[string]string {
	metadata := map[string]string{
		"name":            sat.Name,
		"norad_id":        sat.NORADID,
		"classification":  sat.Classification,
		"source":          "CelesTrak",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
	
	if sat.LaunchYear != "" {
		metadata["launch_year"] = sat.LaunchYear
	}
	if sat.LaunchNumber != "" {
		metadata["launch_number"] = sat.LaunchNumber
	}
	if sat.LaunchPiece != "" {
		metadata["launch_piece"] = sat.LaunchPiece
	}
	
	// Orbital parameters
	metadata["inclination"] = fmt.Sprintf("%.2f", sat.Inclination)
	metadata["raan"] = fmt.Sprintf("%.2f", sat.RAAN)
	metadata["eccentricity"] = fmt.Sprintf("%.6f", sat.Eccentricity)
	metadata["mean_motion"] = fmt.Sprintf("%.4f", sat.MeanMotion)
	
	// Determine orbit type
	if sat.Inclination > 80 {
		metadata["orbit_type"] = "polar"
	} else if sat.MeanMotion > 12 {
		metadata["orbit_type"] = "leo"
	} else if sat.MeanMotion > 2 {
		metadata["orbit_type"] = "meo"
	} else {
		metadata["orbit_type"] = "geo"
	}
	
	// Satellite type detection
	nameLower := strings.ToLower(sat.Name)
	if strings.Contains(nameLower, "starlink") {
		metadata["satellite_type"] = "starlink"
	} else if strings.Contains(nameLower, "gps") {
		metadata["satellite_type"] = "gps"
	} else if strings.Contains(nameLower, "iss") || strings.Contains(nameLower, "station") {
		metadata["satellite_type"] = "space_station"
	} else if strings.Contains(nameLower, "weather") {
		metadata["satellite_type"] = "weather"
	} else if strings.Contains(nameLower, "communications") || strings.Contains(nameLower, "comms") {
		metadata["satellite_type"] = "communications"
	} else {
		metadata["satellite_type"] = "other"
	}
	
	return metadata
}

// generateBadges creates badges for the satellite event
func (p *CelesTrakProvider) generateBadges(sat *Satellite) []model.Badge {
	timestamp := time.Now().UTC()
	badges := []model.Badge{
		{
			Label:     "CelesTrak",
			Type:      "source",
			Timestamp: timestamp,
		},
		{
			Label:     "Satellite",
			Type:      "object_type",
			Timestamp: timestamp,
		},
		{
			Label:     "NORAD " + sat.NORADID,
			Type:      "identifier",
			Timestamp: timestamp,
		},
	}
	
	// Add classification badge
	if sat.Classification != "" && sat.Classification != "U" {
		badges = append(badges, model.Badge{
			Label:     "Class: " + sat.Classification,
			Type:      "classification",
			Timestamp: timestamp,
		})
	}
	
	// Add orbit type badge
	if sat.Inclination > 80 {
		badges = append(badges, model.Badge{
			Label:     "Polar Orbit",
			Type:      "orbit",
			Timestamp: timestamp,
		})
	} else if sat.MeanMotion > 12 {
		badges = append(badges, model.Badge{
			Label:     "LEO",
			Type:      "orbit",
			Timestamp: timestamp,
		})
	} else if sat.MeanMotion > 2 {
		badges = append(badges, model.Badge{
			Label:     "MEO",
			Type:      "orbit",
			Timestamp: timestamp,
		})
	} else {
		badges = append(badges, model.Badge{
			Label:     "GEO",
			Type:      "orbit",
			Timestamp: timestamp,
		})
	}
	
	// Add satellite type badge
	nameLower := strings.ToLower(sat.Name)
	if strings.Contains(nameLower, "starlink") {
		badges = append(badges, model.Badge{
			Label:     "Starlink",
			Type:      "constellation",
			Timestamp: timestamp,
		})
	} else if strings.Contains(nameLower, "gps") {
		badges = append(badges, model.Badge{
			Label:     "GPS",
			Type:      "navigation",
			Timestamp: timestamp,
		})
	} else if strings.Contains(nameLower, "iss") || strings.Contains(nameLower, "station") {
		badges = append(badges, model.Badge{
			Label:     "Space Station",
			Type:      "manned",
			Timestamp: timestamp,
		})
	} else if strings.Contains(nameLower, "weather") {
		badges = append(badges, model.Badge{
			Label:     "Weather",
			Type:      "observation",
			Timestamp: timestamp,
		})
	}
	
	return badges
}

// RSS structures for parsing (used by other providers)

