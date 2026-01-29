package environment

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheDir(t *testing.T) {
	result := cacheDir("/home/user/project")
	expected := "/home/user/project/.lazytf"
	if result != expected {
		t.Errorf("cacheDir() = %q; want %q", result, expected)
	}
}

func TestPreferenceFilePath(t *testing.T) {
	baseDir := filepath.Join("home", "user", "project")
	result := preferenceFilePath(baseDir)
	expected := filepath.Join(baseDir, ".lazytf", envConfigFileName)
	if result != expected {
		t.Errorf("preferenceFilePath() = %q; want %q", result, expected)
	}
}

func TestFilterPreferenceFilePath(t *testing.T) {
	tests := []struct {
		name      string
		baseDir   string
		workspace string
		contains  string
	}{
		{"normal workspace", "/proj", "dev", "dev.json"},
		{"workspace with slash", "/proj", "env/staging", "env_staging.json"},
		{"workspace with backslash", "/proj", "env\\prod", "env_prod.json"},
		{"workspace with colon", "/proj", "C:test", "C_test.json"},
		{"empty workspace", "/proj", "", "default.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterPreferenceFilePath(tt.baseDir, tt.workspace)
			if !contains(result, tt.contains) {
				t.Errorf("filterPreferenceFilePath(%q, %q) = %q; expected to contain %q",
					tt.baseDir, tt.workspace, result, tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[len(s)-len(substr):] == substr || filepath.Base(s) == substr)
}

func TestLoadPreference(t *testing.T) {
	// Test with empty base dir
	t.Run("empty base dir", func(t *testing.T) {
		_, err := LoadPreference("")
		if err == nil {
			t.Error("expected error for empty base dir")
		}
	})

	t.Run("whitespace base dir", func(t *testing.T) {
		_, err := LoadPreference("   ")
		if err == nil {
			t.Error("expected error for whitespace base dir")
		}
	})

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := LoadPreference(tmpDir)
		if !errors.Is(err, ErrNoPreference) {
			t.Errorf("expected ErrNoPreference, got %v", err)
		}
	})

	// Test with valid file
	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheDir := filepath.Join(tmpDir, ".lazytf")
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			t.Fatal(err)
		}

		pref := Preference{
			Strategy:    StrategyWorkspace,
			Environment: "production",
			UpdatedAt:   time.Now(),
		}
		data, _ := json.Marshal(pref)
		path := filepath.Join(cacheDir, envConfigFileName)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		loaded, err := LoadPreference(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaded.Strategy != StrategyWorkspace {
			t.Errorf("expected strategy %v, got %v", StrategyWorkspace, loaded.Strategy)
		}
		if loaded.Environment != "production" {
			t.Errorf("expected environment 'production', got %q", loaded.Environment)
		}
	})

	// Test with invalid JSON
	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheDir := filepath.Join(tmpDir, ".lazytf")
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(cacheDir, envConfigFileName)
		if err := os.WriteFile(path, []byte("{invalid json}"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := LoadPreference(tmpDir)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	// Test with empty preference
	t.Run("empty preference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheDir := filepath.Join(tmpDir, ".lazytf")
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			t.Fatal(err)
		}

		pref := Preference{} // Empty
		data, _ := json.Marshal(pref)
		path := filepath.Join(cacheDir, envConfigFileName)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := LoadPreference(tmpDir)
		if !errors.Is(err, ErrNoPreference) {
			t.Errorf("expected ErrNoPreference for empty preference, got %v", err)
		}
	})
}

func TestSavePreference(t *testing.T) {
	// Test with empty base dir
	t.Run("empty base dir", func(t *testing.T) {
		err := SavePreference("", Preference{Strategy: StrategyFolder})
		if err == nil {
			t.Error("expected error for empty base dir")
		}
	})

	// Test successful save
	t.Run("successful save", func(t *testing.T) {
		tmpDir := t.TempDir()
		pref := Preference{
			Strategy:    StrategyFolder,
			Environment: "staging",
		}

		err := SavePreference(tmpDir, pref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		path := preferenceFilePath(tmpDir)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}

		var loaded Preference
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("failed to unmarshal saved file: %v", err)
		}

		if loaded.Strategy != StrategyFolder {
			t.Errorf("expected strategy %v, got %v", StrategyFolder, loaded.Strategy)
		}
		if loaded.Environment != "staging" {
			t.Errorf("expected environment 'staging', got %q", loaded.Environment)
		}
		if loaded.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	})

	// Test save with existing UpdatedAt
	t.Run("preserve UpdatedAt", func(t *testing.T) {
		tmpDir := t.TempDir()
		fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		pref := Preference{
			Strategy:  StrategyWorkspace,
			UpdatedAt: fixedTime,
		}

		err := SavePreference(tmpDir, pref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := preferenceFilePath(tmpDir)
		data, _ := os.ReadFile(path)
		var loaded Preference
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if !loaded.UpdatedAt.Equal(fixedTime) {
			t.Errorf("expected UpdatedAt %v, got %v", fixedTime, loaded.UpdatedAt)
		}
	})
}

func TestLoadFilterPreference(t *testing.T) {
	// Test with empty base dir
	t.Run("empty base dir", func(t *testing.T) {
		_, err := LoadFilterPreference("", "default")
		if err == nil {
			t.Error("expected error for empty base dir")
		}
	})

	// Test with non-existent file (should return defaults)
	t.Run("non-existent returns defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		pref, err := LoadFilterPreference(tmpDir, "workspace")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !pref.FilterCreate || !pref.FilterUpdate || !pref.FilterDelete || !pref.FilterReplace {
			t.Error("expected all filters enabled by default")
		}
	})

	// Test with valid file
	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filterDir := filepath.Join(tmpDir, ".lazytf", filtersDirName)
		if err := os.MkdirAll(filterDir, 0o755); err != nil {
			t.Fatal(err)
		}

		pref := FilterPreference{
			FilterCreate:  true,
			FilterUpdate:  false,
			FilterDelete:  true,
			FilterReplace: false,
		}
		data, _ := json.Marshal(pref)
		path := filepath.Join(filterDir, "myworkspace.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		loaded, err := LoadFilterPreference(tmpDir, "myworkspace")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !loaded.FilterCreate {
			t.Error("expected FilterCreate true")
		}
		if loaded.FilterUpdate {
			t.Error("expected FilterUpdate false")
		}
	})

	// Test with invalid JSON
	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		filterDir := filepath.Join(tmpDir, ".lazytf", filtersDirName)
		if err := os.MkdirAll(filterDir, 0o755); err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(filterDir, "badworkspace.json")
		if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := LoadFilterPreference(tmpDir, "badworkspace")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestSaveFilterPreference(t *testing.T) {
	// Test with empty base dir
	t.Run("empty base dir", func(t *testing.T) {
		err := SaveFilterPreference("", "workspace", FilterPreference{})
		if err == nil {
			t.Error("expected error for empty base dir")
		}
	})

	// Test successful save
	t.Run("successful save", func(t *testing.T) {
		tmpDir := t.TempDir()
		pref := FilterPreference{
			FilterCreate:  true,
			FilterUpdate:  false,
			FilterDelete:  true,
			FilterReplace: false,
		}

		err := SaveFilterPreference(tmpDir, "myworkspace", pref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		path := filterPreferenceFilePath(tmpDir, "myworkspace")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("expected file to be created")
		}
	})

	// Test save with special characters in workspace
	t.Run("workspace with special chars", func(t *testing.T) {
		tmpDir := t.TempDir()
		pref := FilterPreference{FilterCreate: true}

		err := SaveFilterPreference(tmpDir, "env/prod/us-east", pref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created with sanitized name
		path := filterPreferenceFilePath(tmpDir, "env/prod/us-east")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("expected file to be created")
		}
	})

	// Test UpdatedAt is set if zero
	t.Run("set UpdatedAt if zero", func(t *testing.T) {
		tmpDir := t.TempDir()
		pref := FilterPreference{FilterCreate: true}

		err := SaveFilterPreference(tmpDir, "workspace", pref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		loaded, _ := LoadFilterPreference(tmpDir, "workspace")
		if loaded.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	})
}

func TestWriteJSONAtomic(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "subdir", "deep", "file.json")

		err := writeJSONAtomic(path, map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("expected file to be created")
		}
	})

	t.Run("writes valid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test.json")
		data := map[string]any{
			"name": "test",
			"num":  42,
		}

		err := writeJSONAtomic(path, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(path)
		var loaded map[string]any
		if err := json.Unmarshal(content, &loaded); err != nil {
			t.Fatalf("failed to parse written JSON: %v", err)
		}
		if loaded["name"] != "test" {
			t.Errorf("expected name 'test', got %v", loaded["name"])
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "perms.json")

		err := writeJSONAtomic(path, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		// Check file is readable/writable by owner only
		perm := info.Mode().Perm()
		if perm&0o077 != 0 {
			t.Errorf("expected restricted permissions, got %o", perm)
		}
	})
}
