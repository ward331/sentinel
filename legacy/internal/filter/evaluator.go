package filter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// DefaultEvaluator implements RuleEvaluator
type DefaultEvaluator struct {
	geofenceEngine GeofenceEngine
}

// NewDefaultEvaluator creates a new DefaultEvaluator
func NewDefaultEvaluator(geofenceEngine GeofenceEngine) *DefaultEvaluator {
	return &DefaultEvaluator{
		geofenceEngine: geofenceEngine,
	}
}

// EvaluateCondition evaluates a single condition against an event
func (e *DefaultEvaluator) EvaluateCondition(ctx context.Context, event *model.Event, condition *Condition) (bool, error) {
	// Get field value from event
	value, err := e.getFieldValue(event, condition.Field)
	if err != nil {
		return false, err
	}

	// Apply operator
	result, err := e.applyOperator(value, condition.Value, condition.Operator)
	if err != nil {
		return false, err
	}

	// Apply negation if needed
	if condition.Negate {
		result = !result
	}

	return result, nil
}

// getFieldValue extracts a field value from an event
func (e *DefaultEvaluator) getFieldValue(event *model.Event, field string) (interface{}, error) {
	switch strings.ToLower(field) {
	case "category":
		return string(event.Category), nil
	case "severity":
		return string(event.Severity), nil
	case "title":
		return event.Title, nil
	case "description":
		return event.Description, nil
	case "source":
		return event.Source, nil
	case "source_id":
		return event.SourceID, nil
	case "magnitude":
		return event.Magnitude, nil
	case "latitude":
		if coords, ok := event.Location.Coordinates.([]float64); ok && len(coords) >= 2 {
			return coords[1], nil // GeoJSON: [lon, lat]
		}
		return nil, nil
	case "longitude":
		if coords, ok := event.Location.Coordinates.([]float64); ok && len(coords) >= 1 {
			return coords[0], nil // GeoJSON: [lon, lat]
		}
		return nil, nil
	case "occurred_at":
		return event.OccurredAt, nil
	case "ingested_at":
		return event.IngestedAt, nil
	case "metadata":
		// For metadata fields, use dot notation: metadata.key
		if strings.HasPrefix(field, "metadata.") {
			key := strings.TrimPrefix(field, "metadata.")
			if val, ok := event.Metadata[key]; ok {
				return val, nil
			}
			return nil, nil
		}
		return event.Metadata, nil
	case "badges":
		return event.Badges, nil
	default:
		// Check if it's a metadata field without prefix
		if val, ok := event.Metadata[field]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("unknown field: %s", field)
	}
}

// applyOperator applies an operator to compare values
func (e *DefaultEvaluator) applyOperator(left, right interface{}, operator string) (bool, error) {
	if left == nil || right == nil {
		// Handle nil comparisons
		switch operator {
		case "eq", "=":
			return left == right, nil
		case "ne", "!=":
			return left != right, nil
		case "is_null":
			return left == nil, nil
		case "is_not_null":
			return left != nil, nil
		default:
			return false, fmt.Errorf("operator %s not supported for nil values", operator)
		}
	}

	// Type-specific comparisons
	switch l := left.(type) {
	case string:
		r, ok := right.(string)
		if !ok {
			// Try to convert right to string
			r = fmt.Sprintf("%v", right)
		}
		return e.compareStrings(l, r, operator)
	case float64:
		r, ok := right.(float64)
		if !ok {
			// Try to convert
			if rf, ok := right.(int); ok {
				r = float64(rf)
			} else if rf, ok := right.(int64); ok {
				r = float64(rf)
			} else {
				return false, fmt.Errorf("cannot compare float64 with %T", right)
			}
		}
		return e.compareNumbers(l, r, operator)
	case int:
		r, ok := right.(int)
		if !ok {
			// Try to convert
			if rf, ok := right.(float64); ok {
				return e.compareNumbers(float64(l), rf, operator)
			} else if rf, ok := right.(int64); ok {
				return e.compareNumbers(float64(l), float64(rf), operator)
			} else {
				return false, fmt.Errorf("cannot compare int with %T", right)
			}
		}
		return e.compareNumbers(float64(l), float64(r), operator)
	case time.Time:
		r, ok := right.(time.Time)
		if !ok {
			// Try to parse string
			if rs, ok := right.(string); ok {
				parsed, err := time.Parse(time.RFC3339, rs)
				if err != nil {
					return false, fmt.Errorf("cannot parse time: %v", err)
				}
				r = parsed
			} else {
				return false, fmt.Errorf("cannot compare time.Time with %T", right)
			}
		}
		return e.compareTimes(l, r, operator)
	case []string:
		// For array comparisons (like badges)
		switch operator {
		case "contains", "in":
			// Check if right is in left array
			if rs, ok := right.(string); ok {
				for _, item := range l {
					if item == rs {
						return true, nil
					}
				}
				return false, nil
			}
			return false, fmt.Errorf("cannot compare []string with %T for operator %s", right, operator)
		case "not_contains", "not_in":
			if rs, ok := right.(string); ok {
				for _, item := range l {
					if item == rs {
						return false, nil
					}
				}
				return true, nil
			}
			return false, fmt.Errorf("cannot compare []string with %T for operator %s", right, operator)
		default:
			return false, fmt.Errorf("operator %s not supported for []string", operator)
		}
	case map[string]string:
		// For metadata map
		switch operator {
		case "has_key":
			if key, ok := right.(string); ok {
				_, exists := l[key]
				return exists, nil
			}
			return false, fmt.Errorf("has_key requires string key")
		case "has_value":
			if value, ok := right.(string); ok {
				for _, v := range l {
					if v == value {
						return true, nil
					}
				}
				return false, nil
			}
			return false, fmt.Errorf("has_value requires string value")
		default:
			return false, fmt.Errorf("operator %s not supported for map[string]string", operator)
		}
	default:
		return false, fmt.Errorf("unsupported type for comparison: %T", left)
	}
}

// compareStrings compares two strings
func (e *DefaultEvaluator) compareStrings(left, right, operator string) (bool, error) {
	switch operator {
	case "eq", "=", "equals":
		return left == right, nil
	case "ne", "!=", "not_equals":
		return left != right, nil
	case "contains":
		return strings.Contains(strings.ToLower(left), strings.ToLower(right)), nil
	case "not_contains":
		return !strings.Contains(strings.ToLower(left), strings.ToLower(right)), nil
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(left), strings.ToLower(right)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(left), strings.ToLower(right)), nil
	case "matches", "regex":
		// Simple substring match for now
		return strings.Contains(strings.ToLower(left), strings.ToLower(right)), nil
	case "in":
		// Check if left is in right (comma-separated list)
		items := strings.Split(right, ",")
		for _, item := range items {
			if strings.TrimSpace(item) == left {
				return true, nil
			}
		}
		return false, nil
	case "not_in":
		items := strings.Split(right, ",")
		for _, item := range items {
			if strings.TrimSpace(item) == left {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported string operator: %s", operator)
	}
}

// compareNumbers compares two numbers
func (e *DefaultEvaluator) compareNumbers(left, right float64, operator string) (bool, error) {
	switch operator {
	case "eq", "=", "equals":
		return left == right, nil
	case "ne", "!=", "not_equals":
		return left != right, nil
	case "gt", ">", "greater_than":
		return left > right, nil
	case "gte", ">=", "greater_than_or_equal":
		return left >= right, nil
	case "lt", "<", "less_than":
		return left < right, nil
	case "lte", "<=", "less_than_or_equal":
		return left <= right, nil
	case "between":
		// Right should be a comma-separated range "min,max"
		if rs, ok := interface{}(right).(string); ok {
			parts := strings.Split(rs, ",")
			if len(parts) != 2 {
				return false, fmt.Errorf("between requires 'min,max' format")
			}
			var min, max float64
			if _, err := fmt.Sscanf(parts[0], "%f", &min); err != nil {
				return false, fmt.Errorf("invalid min value: %v", err)
			}
			if _, err := fmt.Sscanf(parts[1], "%f", &max); err != nil {
				return false, fmt.Errorf("invalid max value: %v", err)
			}
			return left >= min && left <= max, nil
		}
		return false, fmt.Errorf("between requires string range")
	case "not_between":
		if rs, ok := interface{}(right).(string); ok {
			parts := strings.Split(rs, ",")
			if len(parts) != 2 {
				return false, fmt.Errorf("not_between requires 'min,max' format")
			}
			var min, max float64
			if _, err := fmt.Sscanf(parts[0], "%f", &min); err != nil {
				return false, fmt.Errorf("invalid min value: %v", err)
			}
			if _, err := fmt.Sscanf(parts[1], "%f", &max); err != nil {
				return false, fmt.Errorf("invalid max value: %v", err)
			}
			return left < min || left > max, nil
		}
		return false, fmt.Errorf("not_between requires string range")
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}

// compareTimes compares two times
func (e *DefaultEvaluator) compareTimes(left, right time.Time, operator string) (bool, error) {
	switch operator {
	case "eq", "=", "equals":
		return left.Equal(right), nil
	case "ne", "!=", "not_equals":
		return !left.Equal(right), nil
	case "gt", ">", "after":
		return left.After(right), nil
	case "gte", ">=", "after_or_equal":
		return left.After(right) || left.Equal(right), nil
	case "lt", "<", "before":
		return left.Before(right), nil
	case "lte", "<=", "before_or_equal":
		return left.Before(right) || left.Equal(right), nil
	case "within":
		// Right should be duration string like "24h", "7d"
		if rs, ok := interface{}(right).(string); ok {
			duration, err := time.ParseDuration(rs)
			if err != nil {
				// Try days format
				if strings.HasSuffix(rs, "d") {
					days := strings.TrimSuffix(rs, "d")
					var d int
					if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
						return false, fmt.Errorf("invalid duration: %v", err)
					}
					duration = time.Duration(d) * 24 * time.Hour
				} else {
					return false, fmt.Errorf("invalid duration: %v", err)
				}
			}
			cutoff := time.Now().Add(-duration)
			return left.After(cutoff), nil
		}
		return false, fmt.Errorf("within requires duration string")
	default:
		return false, fmt.Errorf("unsupported time operator: %s", operator)
	}
}

// ValidateCondition validates a condition configuration
func (e *DefaultEvaluator) ValidateCondition(condition *Condition) error {
	if condition.Field == "" {
		return fmt.Errorf("field is required")
	}
	if condition.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	
	// Check if field is supported
	supported := e.SupportedFields()
	found := false
	for _, field := range supported {
		if field == condition.Field {
			found = true
			break
		}
	}
	if !found && !strings.HasPrefix(condition.Field, "metadata.") {
		// Allow any metadata field
		return fmt.Errorf("unsupported field: %s", condition.Field)
	}
	
	// Check if operator is supported for this field
	operators := e.SupportedOperators(condition.Field)
	if len(operators) > 0 {
		found = false
		for _, op := range operators {
			if op == condition.Operator {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("operator %s not supported for field %s", condition.Operator, condition.Field)
		}
	}
	
	return nil
}

// SupportedFields returns list of fields that can be filtered
func (e *DefaultEvaluator) SupportedFields() []string {
	return []string{
		"category",
		"severity",
		"title",
		"description",
		"source",
		"source_id",
		"magnitude",
		"confidence",
		"latitude",
		"longitude",
		"country",
		"region",
		"created_at",
		"updated_at",
		"metadata",
		"badges",
	}
}

// SupportedOperators returns operators supported for each field
func (e *DefaultEvaluator) SupportedOperators(field string) []string {
	switch field {
	case "category", "severity", "title", "description", "source", "source_id", "country", "region":
		return []string{"eq", "ne", "contains", "not_contains", "starts_with", "ends_with", "matches", "in", "not_in"}
	case "magnitude", "confidence":
		return []string{"eq", "ne", "gt", "gte", "lt", "lte", "between", "not_between"}
	case "latitude", "longitude":
		return []string{"eq", "ne", "gt", "gte", "lt", "lte", "between", "not_between"}
	case "created_at", "updated_at":
		return []string{"eq", "ne", "gt", "gte", "lt", "lte", "within"}
	case "badges":
		return []string{"contains", "not_contains", "in", "not_in"}
	case "metadata":
		return []string{"has_key", "has_value"}
	default:
		if strings.HasPrefix(field, "metadata.") {
			// Metadata fields can use most operators
			return []string{"eq", "ne", "contains", "not_contains", "starts_with", "ends_with", "matches", "in", "not_in", "gt", "gte", "lt", "lte", "between", "not_between"}
		}
		return []string{}
	}
}