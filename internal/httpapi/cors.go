package httpapi

import (
	"net/http"
	"regexp"
	"sort"
	"strings"

	"ble-printer-bridge/internal/config"
	"ble-printer-bridge/internal/logging"
)

const (
	corsAllowMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	corsAllowHeaders = "content-type,authorization,x-api-key"
	corsMaxAge       = "600"
)

type corsConfig struct {
	allowOrigins        map[string]struct{}
	allowOriginPatterns []originPattern
}

type originPattern struct {
	raw   string
	re    *regexp.Regexp
	valid bool
}

func newCORSConfig(cfg *config.Config, log *logging.Logger) *corsConfig {
	allowOrigins := parseCSV(cfg.CORS.AllowOrigins)
	allowOriginPatterns := parsePatterns(cfg.CORS.AllowOriginPatterns, log)

	originsList := make([]string, 0, len(allowOrigins))
	for origin := range allowOrigins {
		originsList = append(originsList, origin)
	}
	sort.Strings(originsList)
	log.Info("cors allow origins: %v", originsList)
	if len(allowOriginPatterns) == 0 {
		log.Info("cors allow origin patterns: []")
	} else {
		for _, pattern := range allowOriginPatterns {
			log.Info("cors allow origin pattern: %q compiled=%t", pattern.raw, pattern.valid)
		}
	}

	return &corsConfig{
		allowOrigins:        allowOrigins,
		allowOriginPatterns: allowOriginPatterns,
	}
}

func parseCSV(value string) map[string]struct{} {
	items := strings.Split(value, ",")
	result := make(map[string]struct{})
	for _, item := range items {
		trimmed := normalizeOrigin(strings.TrimSpace(item))
		if trimmed == "" {
			continue
		}
		result[trimmed] = struct{}{}
	}
	return result
}

func parsePatterns(value string, log *logging.Logger) []originPattern {
	items := strings.Split(value, ",")
	patterns := make([]originPattern, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		re, err := regexp.Compile(toRegexPattern(trimmed))
		if err != nil {
			log.Warn("cors origin pattern compile failed: %q error=%v", trimmed, err)
			patterns = append(patterns, originPattern{raw: trimmed, valid: false})
			continue
		}
		patterns = append(patterns, originPattern{raw: trimmed, re: re, valid: true})
	}
	return patterns
}

func toRegexPattern(pattern string) string {
	if strings.Contains(pattern, "*") {
		escaped := regexp.QuoteMeta(pattern)
		wildcard := strings.ReplaceAll(escaped, "\\*", ".*")
		return "^" + wildcard + "$"
	}
	return pattern
}

func isOriginAllowed(cfg *corsConfig, origin string) (bool, string) {
	normalizedOrigin := normalizeOrigin(origin)
	if normalizedOrigin == "" {
		return false, "missing origin"
	}
	if _, ok := cfg.allowOrigins[normalizedOrigin]; ok {
		return true, "explicit allowlist"
	}
	for _, pattern := range cfg.allowOriginPatterns {
		if !pattern.valid || pattern.re == nil {
			continue
		}
		if pattern.re.MatchString(normalizedOrigin) {
			return true, "pattern match"
		}
	}
	return false, "not allowed"
}

func normalizeOrigin(origin string) string {
	trimmed := strings.TrimSpace(origin)
	trimmed = strings.TrimSuffix(trimmed, "/")
	return trimmed
}

func corsMiddleware(log *logging.Logger, cfg *corsConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed, reason := isOriginAllowed(cfg, origin)
		if origin != "" && !allowed {
			log.Debug("cors reject origin %q: %s", origin, reason)
		}

		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if origin != "" && allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
			w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
