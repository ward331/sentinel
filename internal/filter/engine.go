package filter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
	"github.com/openclaw/sentinel-backend/internal/storage"
)

// DefaultEngine implements FilterEngine
type DefaultEngine struct {
	storage       storage.Storage
	evaluator     RuleEvaluator
	geofenceEngine GeofenceEngine
	filters       map[string]*Filter
	mu            sync.RWMutex
	stats         map[string]int64 // filterID -> match count
}

// NewDefaultEngine creates a new DefaultEngine
func NewDefaultEngine(storage storage.Storage, evaluator RuleEvaluator, geofenceEngine GeofenceEngine) (*DefaultEngine, error) {
	engine := &DefaultEngine{
		storage:       storage,
		evaluator:     evaluator,
		geofenceEngine: geofenceEngine,
		filters:       make(map[string]*Filter),
		stats:         make(map[string]int64),
	}

	// Load filters from storage
	if err := engine.loadFilters(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load filters: %v", err)
	}

	return engine, nil
}

// loadFilters loads all filters from storage
func (e *DefaultEngine) loadFilters(ctx context.Context) error {
	// This would load from database
	// For now, initialize with empty map
	e.filters = make(map[string]*Filter)
	e.stats = make(map[string]int64)
	return nil
}

// Evaluate evaluates a single event against all active filters
func (e *DefaultEngine) Evaluate(ctx context.Context, event *model.Event) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var matchingFilters []string

	for filterID, filter := range e.filters {
		if !filter.Enabled {
			continue
		}

		matches, err := e.evaluateFilter(ctx, event, filter)
		if err != nil {
			return nil, fmt.Errorf("error evaluating filter %s: %v", filterID, err)
		}

		if matches {
			matchingFilters = append(matchingFilters, filterID)
			
			// Update stats
			e.stats[filterID]++
			
			// Execute actions
			if err := e.executeActions(ctx, event, filter); err != nil {
				// Log error but don't fail evaluation
				fmt.Printf("Error executing actions for filter %s: %v\n", filterID, err)
			}
		}
	}

	return matchingFilters, nil
}

// evaluateFilter evaluates an event against a single filter
func (e *DefaultEngine) evaluateFilter(ctx context.Context, event *model.Event, filter *Filter) (bool, error) {
	if len(filter.Conditions) == 0 {
		// Empty filter matches everything
		return true, nil
	}

	// All conditions must match (AND logic)
	for _, condition := range filter.Conditions {
		matches, err := e.evaluator.EvaluateCondition(ctx, event, &condition)
		if err != nil {
			return false, fmt.Errorf("error evaluating condition: %v", err)
		}

		if !matches {
			return false, nil
		}
	}

	return true, nil
}

// executeActions executes all actions for a filter
func (e *DefaultEngine) executeActions(ctx context.Context, event *model.Event, filter *Filter) error {
	for _, action := range filter.Actions {
		if !action.Enabled {
			continue
		}

		switch action.Type {
		case "notify":
			if err := e.executeNotifyAction(ctx, event, action.Config); err != nil {
				return fmt.Errorf("error executing notify action: %v", err)
			}
		case "tag":
			if err := e.executeTagAction(ctx, event, action.Config); err != nil {
				return fmt.Errorf("error executing tag action: %v", err)
			}
		case "route":
			if err := e.executeRouteAction(ctx, event, action.Config); err != nil {
				return fmt.Errorf("error executing route action: %v", err)
			}
		case "log":
			if err := e.executeLogAction(ctx, event, action.Config); err != nil {
				return fmt.Errorf("error executing log action: %v", err)
			}
		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}

	return nil
}

// executeNotifyAction executes a notification action
func (e *DefaultEngine) executeNotifyAction(ctx context.Context, event *model.Event, config map[string]interface{}) error {
	// This would integrate with notification system
	// For now, just log
	fmt.Printf("NOTIFY: Event %s matched filter, would send notification\n", event.ID)
	return nil
}

// executeTagAction executes a tagging action
func (e *DefaultEngine) executeTagAction(ctx context.Context, event *model.Event, config map[string]interface{}) error {
	// Add tags to event metadata
	tag, ok := config["tag"].(string)
	if !ok {
		return fmt.Errorf("tag action requires 'tag' configuration")
	}

	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}

	// Add filter tag
	event.Metadata["filter_tag"] = tag
	
	// Also add to badges if not already present
	found := false
	for _, badge := range event.Badges {
		if badge.Label == tag {
			found = true
			break
		}
	}
	if !found {
		event.Badges = append(event.Badges, model.Badge{
			Label:     tag,
			Type:      model.BadgeTypeFilter,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// executeRouteAction executes a routing action
func (e *DefaultEngine) executeRouteAction(ctx context.Context, event *model.Event, config map[string]interface{}) error {
	// Route to different stream or storage
	// For now, just log
	destination, _ := config["destination"].(string)
	fmt.Printf("ROUTE: Event %s would be routed to %s\n", event.ID, destination)
	return nil
}

// executeLogAction executes a logging action
func (e *DefaultEngine) executeLogAction(ctx context.Context, event *model.Event, config map[string]interface{}) error {
	// Log filter match
	message, _ := config["message"].(string)
	if message == "" {
		message = "Event matched filter"
	}
	fmt.Printf("FILTER LOG: %s - Event: %s, Source: %s\n", message, event.ID, event.Source)
	return nil
}

// AddFilter adds a new filter
func (e *DefaultEngine) AddFilter(ctx context.Context, filter *Filter) error {
	// Validate filter
	if filter.ID == "" {
		filter.ID = generateID()
	}
	if filter.Name == "" {
		return fmt.Errorf("filter name is required")
	}

	// Validate all conditions
	for i := range filter.Conditions {
		if err := e.evaluator.ValidateCondition(&filter.Conditions[i]); err != nil {
			return fmt.Errorf("invalid condition %d: %v", i, err)
		}
	}

	// Set timestamps
	now := time.Now().UTC()
	if filter.CreatedAt.IsZero() {
		filter.CreatedAt = now
	}
	filter.UpdatedAt = now

	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if filter already exists
	if _, exists := e.filters[filter.ID]; exists {
		return fmt.Errorf("filter with ID %s already exists", filter.ID)
	}

	// Add to memory
	e.filters[filter.ID] = filter
	e.stats[filter.ID] = 0

	// Save to storage
	if err := e.saveFilter(ctx, filter); err != nil {
		// Rollback
		delete(e.filters, filter.ID)
		delete(e.stats, filter.ID)
		return fmt.Errorf("failed to save filter: %v", err)
	}

	return nil
}

// UpdateFilter updates an existing filter
func (e *DefaultEngine) UpdateFilter(ctx context.Context, id string, filter *Filter) error {
	if id == "" {
		return fmt.Errorf("filter ID is required")
	}

	// Validate filter
	if filter.Name == "" {
		return fmt.Errorf("filter name is required")
	}

	// Validate all conditions
	for i := range filter.Conditions {
		if err := e.evaluator.ValidateCondition(&filter.Conditions[i]); err != nil {
			return fmt.Errorf("invalid condition %d: %v", i, err)
		}
	}

	// Set updated timestamp
	filter.UpdatedAt = time.Now().UTC()
	filter.ID = id // Ensure ID matches

	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if filter exists
	oldFilter, exists := e.filters[id]
	if !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}

	// Preserve created timestamp
	if filter.CreatedAt.IsZero() {
		filter.CreatedAt = oldFilter.CreatedAt
	}

	// Update in memory
	e.filters[id] = filter

	// Save to storage
	if err := e.saveFilter(ctx, filter); err != nil {
		// Rollback
		e.filters[id] = oldFilter
		return fmt.Errorf("failed to update filter: %v", err)
	}

	return nil
}

// RemoveFilter removes a filter
func (e *DefaultEngine) RemoveFilter(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("filter ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if filter exists
	if _, exists := e.filters[id]; !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}

	// Remove from storage
	if err := e.deleteFilter(ctx, id); err != nil {
		return fmt.Errorf("failed to delete filter: %v", err)
	}

	// Remove from memory
	delete(e.filters, id)
	delete(e.stats, id)

	return nil
}

// GetFilter retrieves a filter by ID
func (e *DefaultEngine) GetFilter(ctx context.Context, id string) (*Filter, error) {
	if id == "" {
		return nil, fmt.Errorf("filter ID is required")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	filter, exists := e.filters[id]
	if !exists {
		return nil, fmt.Errorf("filter with ID %s not found", id)
	}

	// Return a copy to prevent modification
	return copyFilter(filter), nil
}

// ListFilters lists all filters
func (e *DefaultEngine) ListFilters(ctx context.Context) ([]*Filter, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filters := make([]*Filter, 0, len(e.filters))
	for _, filter := range e.filters {
		filters = append(filters, copyFilter(filter))
	}

	return filters, nil
}

// EnableFilter enables a filter
func (e *DefaultEngine) EnableFilter(ctx context.Context, id string) error {
	return e.setFilterEnabled(ctx, id, true)
}

// DisableFilter disables a filter
func (e *DefaultEngine) DisableFilter(ctx context.Context, id string) error {
	return e.setFilterEnabled(ctx, id, false)
}

// setFilterEnabled sets the enabled state of a filter
func (e *DefaultEngine) setFilterEnabled(ctx context.Context, id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("filter ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	filter, exists := e.filters[id]
	if !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}

	if filter.Enabled == enabled {
		// No change needed
		return nil
	}

	// Update filter
	updatedFilter := copyFilter(filter)
	updatedFilter.Enabled = enabled
	updatedFilter.UpdatedAt = time.Now().UTC()

	// Save to storage
	if err := e.saveFilter(ctx, updatedFilter); err != nil {
		return fmt.Errorf("failed to update filter: %v", err)
	}

	// Update in memory
	e.filters[id] = updatedFilter

	return nil
}

// MatchCount returns statistics about filter matches
func (e *DefaultEngine) MatchCount(ctx context.Context, filterID string) (int64, error) {
	if filterID == "" {
		// Return total matches across all filters
		e.mu.RLock()
		defer e.mu.RUnlock()
		
		var total int64
		for _, count := range e.stats {
			total += count
		}
		return total, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	count, exists := e.stats[filterID]
	if !exists {
		return 0, fmt.Errorf("filter with ID %s not found", filterID)
	}

	return count, nil
}

// saveFilter saves a filter to storage
func (e *DefaultEngine) saveFilter(ctx context.Context, filter *Filter) error {
	// This would save to database
	// For now, just store in memory (already done)
	return nil
}

// deleteFilter deletes a filter from storage
func (e *DefaultEngine) deleteFilter(ctx context.Context, id string) error {
	// This would delete from database
	// For now, just remove from memory (already done)
	return nil
}

// copyFilter creates a deep copy of a filter
func copyFilter(filter *Filter) *Filter {
	copied := &Filter{
		ID:          filter.ID,
		Name:        filter.Name,
		Description: filter.Description,
		Enabled:     filter.Enabled,
		CreatedAt:   filter.CreatedAt,
		UpdatedAt:   filter.UpdatedAt,
	}

	// Copy conditions
	if filter.Conditions != nil {
		copied.Conditions = make([]Condition, len(filter.Conditions))
		copy(copied.Conditions, filter.Conditions)
	}

	// Copy actions
	if filter.Actions != nil {
		copied.Actions = make([]Action, len(filter.Actions))
		copy(copied.Actions, filter.Actions)
	}

	return copied
}

// generateID generates a unique filter ID
func generateID() string {
	return fmt.Sprintf("filter_%d", time.Now().UnixNano())
}