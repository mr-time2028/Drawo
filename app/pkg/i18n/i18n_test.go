package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestI18nEngine(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "locales-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	faData := `{
		"meta": {"direction": "rtl", "font": "B Nazanin"},
		"game": {"welcome": "سلام"}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "fa.json"), []byte(faData), 0644))

	// Test Init
	err = Init(tempDir, "fa")
	require.NoError(t, err)

	// Test Translation
	assert.Equal(t, "سلام", T("fa", "game.welcome"))
	
	// Test Fallback
	assert.Equal(t, "سلام", T("unknown", "game.welcome"))

	// Test Missing Key
	assert.Equal(t, "game.ghost", T("fa", "game.ghost"))

	// Test Direction
	assert.Equal(t, "rtl", GetDirection("fa"))
}
