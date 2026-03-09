# OSINT Resources & Presets

## Pre-loaded Resources

### Iran Conflict Tracking

**Category**: Conflict Tracking  
**Alert Tier**: TIER 3  
**Priority**: High  
**Update Frequency**: 15-30 minutes  

#### Sources:

1. **OSINT Dataset (GitHub)**
   - URL: `https://raw.githubusercontent.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data/main/data/waves.json`
   - Data: Operation name, date, weapons, targets, coordinates, interception rate, impact assessment
   - Update: Every 15 minutes
   - Parser: `internal/provider/iranconflict.go`

2. **ISW RSS Feed**
   - URL: `https://understandingwar.org/rss.xml`
   - Filter: Iran, Israel, Middle East keywords
   - Update: Every 30 minutes
   - Priority: High (tagged in news aggregator)

3. **Iran Strike Map (Embedded)**
   - URL: `https://www.iranstrikemap.com`
   - Type: Embedded iframe
   - Dimensions: 100% width, 600px height
   - Category: Conflict Tracking

#### Event Processing:
- **Category**: `conflict`
- **Severity**: Based on weapon type and target type
  - Critical: Ballistic/cruise missiles + high-value targets
  - High: High-severity weapons OR high-value targets  
  - Medium: Other conflict events
- **Magnitude**: Calculated from weapon count, interception rate, targets destroyed
- **Badges**: OSINT Conflict Data (dark red), Exact (dark green), Real-time (blue)

#### Metadata Fields:
```json
{
  "wave_number": 1,
  "operation_name": "Operation Name",
  "weapon_type": "ballistic_missile",
  "target_type": "military_base",
  "interception_rate": 85.5,
  "impact_assessment": "moderate",
  "total_weapons": 12,
  "targets_destroyed": 3,
  "alert_tier": "TIER 3",
  "data_source": "Iran-Israel-War-2026-OSINT-Data",
  "conflict_region": "Middle East",
  "primary_actors": ["Iran", "Israel"]
}
```

#### Keywords:
- Iran, Israel, Middle East, Conflict, OSINT
- Missile, Drone, Strike, Defense, Interception
- Ballistic, Cruise, Hypersonic, Swarm
- Military, Infrastructure, Civilian, Government

#### Preset Configuration:
```json
{
  "name": "Iran Conflict Tracking",
  "category": "Conflict Tracking",
  "description": "Real-time tracking of Iran-Israel conflict events",
  "iframe": {
    "src": "https://www.iranstrikemap.com",
    "width": "100%",
    "height": "600px",
    "title": "Iran Strike Map - Real-time conflict tracking"
  },
  "sources": [
    {
      "name": "OSINT Dataset",
      "url": "https://github.com/danielrosehill/Iran-Israel-War-2026-OSINT-Data",
      "description": "Comprehensive OSINT data on Iran-Israel conflict",
      "update_frequency": "15 minutes"
    },
    {
      "name": "ISW RSS Feed",
      "url": "https://understandingwar.org/rss.xml",
      "description": "Institute for the Study of War analysis",
      "update_frequency": "30 minutes"
    },
    {
      "name": "Iran Strike Map",
      "url": "https://www.iranstrikemap.com",
      "description": "Interactive map of strike events",
      "type": "iframe"
    }
  ],
  "keywords": [
    "Iran", "Israel", "Middle East", "Conflict", "OSINT",
    "Missile", "Drone", "Strike", "Defense", "Interception"
  ],
  "alert_tier": "TIER 3",
  "priority": "high"
}
```

#### Integration:
- **Provider**: `IranConflictProvider` in `internal/provider/iranconflict.go`
- **Config**: Enabled by default, 15-minute polling interval
- **Media Wall**: Added as preset under "Conflict Tracking" category
- **Alerts**: TIER 3 alerts for new strike waves
- **Filters**: Available by weapon type, target type, interception rate

#### Data Flow:
1. Poll GitHub dataset every 15 minutes
2. Parse waves.json into structured events
3. Calculate severity and magnitude
4. Add OSINT badges and metadata
5. Store in database with conflict category
6. Broadcast via SSE for real-time updates
7. Display on globe with conflict markers (red/orange)
8. Include in media wall iframe preset

#### Alert Rules (Default):
- New strike wave detected → TIER 3 alert
- High-severity weapon used → Critical alert  
- Low interception rate (< 50%) → Warning alert
- Multiple targets destroyed → High alert

#### Visualization:
- **Globe Markers**: Red/orange conflict markers
- **Media Wall**: Embedded Iran Strike Map iframe
- **Details Panel**: Weapon/target info, interception rate, impact
- **Timeline**: Chronological wave visualization
- **Heatmap**: Strike density overlay