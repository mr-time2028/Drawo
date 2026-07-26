// Package i18n provides high-performance internationalization for Drawo.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Engine manages multiple language bundles in memory.
type Engine struct {
	mu          sync.RWMutex
	bundles     map[string]map[string]interface{}
	directions  map[string]string
	defaultLang string
}

var globalEngine *Engine

// Init loads JSON files from the locales directory.
func Init(localesPath string, defaultLang string) error {
	engine := &Engine{
		bundles:     make(map[string]map[string]interface{}),
		directions:  make(map[string]string),
		defaultLang: defaultLang,
	}

	files, err := os.ReadDir(localesPath)
	if err != nil {
		return fmt.Errorf("read locales dir: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		langCode := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(localesPath, file.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file %s: %w", langCode, err)
		}

		var fullBundle map[string]interface{}
		if err := json.Unmarshal(data, &fullBundle); err != nil {
			return fmt.Errorf("unmarshal %s: %w", langCode, err)
		}

		// Extract metadata for RTL/LTR support
		if meta, ok := fullBundle["meta"].(map[string]interface{}); ok {
			if dir, ok := meta["direction"].(string); ok {
				engine.directions[langCode] = dir
			}
		}

		engine.bundles[langCode] = fullBundle
	}

	globalEngine = engine
	return nil
}

// T returns a translated string for a given key.
// Format: "category.key" (e.g., "errors.not_found")
func T(lang string, key string) string {
	if globalEngine == nil {
		return key
	}

	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()

	bundle, ok := globalEngine.bundles[lang]
	if !ok {
		bundle = globalEngine.bundles[globalEngine.defaultLang]
	}

	return navigateKey(bundle, key)
}

// GetDirection returns "rtl" or "ltr" for a language.
// Useful for the frontend to adjust the layout for Persian.
func GetDirection(lang string) string {
	if globalEngine == nil {
		return "ltr"
	}
	if dir, ok := globalEngine.directions[lang]; ok {
		return dir
	}
	return "ltr"
}

// navigateKey parses "a.b.c" and finds it in the nested JSON map.
func navigateKey(m map[string]interface{}, key string) string {
	parts := strings.Split(key, ".")
	var current interface{} = m

	for _, part := range parts {
		if subMap, ok := current.(map[string]interface{}); ok {
			current = subMap[part]
		} else {
			return key // Key not found
		}
	}

	if val, ok := current.(string); ok {
		return val
	}
	return key
}
