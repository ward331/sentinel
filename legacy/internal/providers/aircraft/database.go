package aircraft

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

//go:embed modes.csv
var modesFS embed.FS

// AircraftInfo contains information about an aircraft
type AircraftInfo struct {
	Hex         string `json:"hex"`
	Registration string `json:"registration"`
	TypeCode    string `json:"type_code"`
	Owner       string `json:"owner"`
	Aircraft    string `json:"aircraft"`
	Military    bool   `json:"military"`
}

// Database manages the aircraft identification database
type Database struct {
	mu      sync.RWMutex
	byHex   map[string]*AircraftInfo
	loaded  bool
	lastLoad time.Time
}

// NewDatabase creates a new aircraft database
func NewDatabase() *Database {
	return &Database{
		byHex:  make(map[string]*AircraftInfo),
		loaded: false,
	}
}

// Load loads the aircraft database from embedded CSV
func (db *Database) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Open embedded CSV file
	file, err := modesFS.Open("modes.csv")
	if err != nil {
		return fmt.Errorf("failed to open modes.csv: %w", err)
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.LazyQuotes = true

	// Skip header
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Clear existing data
	db.byHex = make(map[string]*AircraftInfo)

	// Read records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows but continue
			continue
		}

		// Parse record (expected format: hex,registration,typecode,owner,aircraft)
		if len(record) >= 5 {
			hex := strings.ToUpper(strings.TrimSpace(record[0]))
			if hex == "" {
				continue
			}

			info := &AircraftInfo{
				Hex:          hex,
				Registration: strings.TrimSpace(record[1]),
				TypeCode:     strings.TrimSpace(record[2]),
				Owner:        strings.TrimSpace(record[3]),
				Aircraft:     strings.TrimSpace(record[4]),
				Military:     db.isMilitary(record[3], record[4]),
			}

			db.byHex[hex] = info
		}
	}

	db.loaded = true
	db.lastLoad = time.Now()
	return nil
}

// isMilitary determines if an aircraft is military based on owner and aircraft name
func (db *Database) isMilitary(owner, aircraft string) bool {
	ownerLower := strings.ToLower(owner)
	aircraftLower := strings.ToLower(aircraft)

	// Military indicators in owner field
	militaryOwners := []string{
		"air force",
		"army",
		"navy",
		"marine",
		"military",
		"defense",
		"ministry of defence",
		"usaf",
		"raf",
		"luftwaffe",
		"aeronautica militare",
		"armée de l'air",
	}

	for _, indicator := range militaryOwners {
		if strings.Contains(ownerLower, indicator) {
			return true
		}
	}

	// Military indicators in aircraft name
	militaryAircraft := []string{
		"c-130",
		"c-17",
		"c-5",
		"kc-135",
		"kc-10",
		"e-3",
		"e-8",
		"rc-135",
		"p-8",
		"f-",
		"b-",
		"a-10",
		"ah-64",
		"uh-60",
		"ch-47",
		"v-22",
		"e-2",
		"e-7",
		"global hawk",
		"reaper",
		"predator",
		"rivet joint",
		"jstars",
		"awacs",
		"tanker",
		"transport",
		"gunship",
	}

	for _, indicator := range militaryAircraft {
		if strings.Contains(aircraftLower, indicator) {
			return true
		}
	}

	return false
}

// Lookup looks up aircraft information by hex code
func (db *Database) Lookup(hex string) (*AircraftInfo, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Ensure database is loaded
	if !db.loaded {
		return nil, false
	}

	// Normalize hex (uppercase, no spaces)
	hex = strings.ToUpper(strings.ReplaceAll(hex, " ", ""))

	info, found := db.byHex[hex]
	return info, found
}

// LookupWithFallback looks up aircraft and returns enriched information
func (db *Database) LookupWithFallback(hex, callsign string) map[string]interface{} {
	info, found := db.Lookup(hex)
	
	result := map[string]interface{}{
		"hex":       hex,
		"callsign":  callsign,
		"identified": found,
		"military":  false,
	}

	if found {
		result["registration"] = info.Registration
		result["type_code"] = info.TypeCode
		result["owner"] = info.Owner
		result["aircraft"] = info.Aircraft
		result["military"] = info.Military
		
		// Generate descriptive name
		if info.Aircraft != "" && info.Registration != "" {
			result["display_name"] = fmt.Sprintf("%s %s", info.Aircraft, info.Registration)
		} else if info.Aircraft != "" {
			result["display_name"] = info.Aircraft
		} else if info.Registration != "" {
			result["display_name"] = fmt.Sprintf("Aircraft %s", info.Registration)
		} else {
			result["display_name"] = fmt.Sprintf("Unknown aircraft %s", hex)
		}
	} else {
		// Try to infer from callsign
		if callsign != "" {
			result["display_name"] = fmt.Sprintf("%s (%s)", callsign, hex)
			
			// Check if callsign indicates military
			if strings.HasPrefix(callsign, "RCH") || // USAF cargo
			   strings.HasPrefix(callsign, "SAM") || // US executive transport
			   strings.HasPrefix(callsign, "CNV") || // US Navy cargo
			   strings.HasPrefix(callsign, "RRR") || // RAF
			   strings.HasPrefix(callsign, "ASA") || // US Army
			   strings.HasPrefix(callsign, "DUKE") || // US Army
			   strings.HasPrefix(callsign, "PAT") || // US Army
			   strings.HasPrefix(callsign, "GAF") || // German Air Force
			   strings.HasPrefix(callsign, "CTM") || // French Air Force
			   strings.HasPrefix(callsign, "IAM") {  // Italian Air Force
				result["military"] = true
			}
		} else {
			result["display_name"] = fmt.Sprintf("Unknown aircraft %s", hex)
		}
	}

	return result
}

// Count returns the number of aircraft in the database
func (db *Database) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.byHex)
}

// LastLoad returns when the database was last loaded
func (db *Database) LastLoad() time.Time {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.lastLoad
}

// IsLoaded returns whether the database is loaded
func (db *Database) IsLoaded() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.loaded
}

// Refresh refreshes the database from embedded source
func (db *Database) Refresh() error {
	return db.Load()
}

// AutoRefresh starts automatic monthly refresh
func (db *Database) AutoRefresh() {
	go func() {
		for {
			time.Sleep(30 * 24 * time.Hour) // Monthly
			if err := db.Refresh(); err != nil {
				fmt.Printf("Failed to refresh aircraft database: %v\n", err)
			} else {
				fmt.Printf("Aircraft database refreshed: %d aircraft\n", db.Count())
			}
		}
	}()
}