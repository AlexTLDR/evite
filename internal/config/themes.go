package config

import (
	"os"
	"regexp"
	"strings"
)

// ThemeConfig holds the light and dark theme names
// These are automatically parsed from static/css/input.css
type ThemeConfig struct {
	Light string
	Dark  string
}

// GetThemes returns the current theme configuration by parsing static/css/input.css
// Expected format in input.css: themes: themeName --default, themeName --prefersdark;
// Falls back to fantasy (light) and aqua (dark) if parsing fails
func GetThemes() ThemeConfig {
	// Try to read from input.css
	cssPath := "static/css/input.css"
	if content, err := os.ReadFile(cssPath); err == nil {
		if themes := parseThemesFromCSS(string(content)); themes != nil {
			return *themes
		}
	}

	// Fallback to defaults if parsing fails
	return ThemeConfig{
		Light: "fantasy",
		Dark:  "aqua",
	}
}

// parseThemesFromCSS extracts theme names from the DaisyUI plugin configuration
// Expected format: themes: themeName --default, themeName --prefersdark;
func parseThemesFromCSS(content string) *ThemeConfig {
	// Match the themes line in the @plugin "daisyui" block
	// Pattern: themes: <light-theme> --default, <dark-theme> --prefersdark;
	re := regexp.MustCompile(`themes:\s*([a-zA-Z0-9-]+)\s+--default\s*,\s*([a-zA-Z0-9-]+)\s+--prefersdark`)
	matches := re.FindStringSubmatch(content)

	if len(matches) == 3 {
		return &ThemeConfig{
			Light: strings.TrimSpace(matches[1]),
			Dark:  strings.TrimSpace(matches[2]),
		}
	}

	return nil
}
