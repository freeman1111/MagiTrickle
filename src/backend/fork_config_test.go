package magitrickle

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadForkConfigCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fork.yaml")

	cfg, err := loadForkConfig(path)
	if err != nil {
		t.Fatalf("loadForkConfig failed: %v", err)
	}
	if expected := []string{"noMT"}; !reflect.DeepEqual(cfg.BypassPolicies, expected) {
		t.Errorf("BypassPolicies = %v, want %v", cfg.BypassPolicies, expected)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("default fork config was not written: %v", err)
	}
	if string(data) != defaultForkConfig {
		t.Errorf("written fork config differs from the default template")
	}
}

func TestLoadForkConfigKeepsUserValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fork.yaml")
	for name, tc := range map[string]struct {
		content  string
		expected []string
	}{
		"empty list disables bypass": {"bypassPolicies: []\n", []string{}},
		"custom list":                {"bypassPolicies:\n  - Guests\n  - Policy4\n", []string{"Guests", "Policy4"}},
		"missing key":                {"# nothing\n", nil},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadForkConfig(path)
			if err != nil {
				t.Fatalf("loadForkConfig failed: %v", err)
			}
			if !reflect.DeepEqual(cfg.BypassPolicies, tc.expected) {
				t.Errorf("BypassPolicies = %#v, want %#v", cfg.BypassPolicies, tc.expected)
			}
		})
	}

	if err := os.WriteFile(path, []byte("bypassPolicies: [\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadForkConfig(path); err == nil {
		t.Errorf("invalid yaml must return an error")
	}
}
