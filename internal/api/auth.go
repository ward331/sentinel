package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled       bool
	APIKeys       []string
	JWTSecret     string
	JWTExpiration time.Duration
	ExemptPaths   []string
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:       false,
		APIKeys:       []string{},
		JWTSecret:     "change-me-in-production",
		JWTExpiration: 24 * time.Hour,
		ExemptPaths: []string{
			"/api/health",
			"/api/metrics",
			"/api/events/stream", // SSE streams don't support auth headers easily
		},
	}
}

// AuthMiddleware creates authentication middleware based on configuration
func AuthMiddleware(config AuthConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		// Authentication disabled, return pass-through middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is exempt
			for _, exemptPath := range config.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Try API key authentication first
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				for _, validKey := range config.APIKeys {
					if apiKey == validKey {
						// Valid API key found
						ctx := context.WithValue(r.Context(), "auth_method", "api_key")
						ctx = context.WithValue(ctx, "api_key", apiKey)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			// Try JWT authentication
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Check if it's a Bearer token
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString := parts[1]
					
					// Parse and validate JWT
					token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
						// Validate signing method
						if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, jwt.ErrSignatureInvalid
						}
						return []byte(config.JWTSecret), nil
					})

					if err == nil && token.Valid {
						// Valid JWT found
						if claims, ok := token.Claims.(jwt.MapClaims); ok {
							ctx := context.WithValue(r.Context(), "auth_method", "jwt")
							ctx = context.WithValue(ctx, "jwt_claims", claims)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			// No valid authentication found
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

// GenerateJWT generates a new JWT token
func GenerateJWT(secret string, expiration time.Duration, claims map[string]interface{}) (string, error) {
	// Create token with claims
	token := jwt.New(jwt.SigningMethodHS256)
	
	// Set standard claims
	tokenClaims := token.Claims.(jwt.MapClaims)
	tokenClaims["exp"] = time.Now().Add(expiration).Unix()
	tokenClaims["iat"] = time.Now().Unix()
	
	// Add custom claims
	for key, value := range claims {
		tokenClaims[key] = value
	}
	
	// Sign token
	return token.SignedString([]byte(secret))
}

// ValidateAPIKey validates an API key against the configuration
func ValidateAPIKey(config AuthConfig, apiKey string) bool {
	if !config.Enabled {
		return true // Authentication disabled
	}
	
	for _, validKey := range config.APIKeys {
		if apiKey == validKey {
			return true
		}
	}
	return false
}

// GetAuthMethod returns the authentication method from context
func GetAuthMethod(ctx context.Context) string {
	if method, ok := ctx.Value("auth_method").(string); ok {
		return method
	}
	return ""
}

// GetAPIKey returns the API key from context
func GetAPIKey(ctx context.Context) string {
	if apiKey, ok := ctx.Value("api_key").(string); ok {
		return apiKey
	}
	return ""
}

// GetJWTClaims returns the JWT claims from context
func GetJWTClaims(ctx context.Context) jwt.MapClaims {
	if claims, ok := ctx.Value("jwt_claims").(jwt.MapClaims); ok {
		return claims
	}
	return nil
}