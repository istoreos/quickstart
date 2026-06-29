package uci

import (
	"errors"
	"reflect"
	"testing"
)

func TestListOptionReturnsCopiedValues(t *testing.T) {
	originalLoadConfig := loadConfig
	originalGet := get
	defer func() {
		loadConfig = originalLoadConfig
		get = originalGet
	}()

	loadConfig = func(config string, force bool) error {
		if config != "quickstart" || !force {
			t.Fatalf("loadConfig called with config=%q force=%v", config, force)
		}
		return nil
	}
	source := []string{"smart", "raid"}
	get = func(config string, section string, option string) ([]string, bool) {
		if config != "quickstart" || section != "modules" || option != "module" {
			t.Fatalf("get called with %q %q %q", config, section, option)
		}
		return source, true
	}

	got := ListOption("quickstart", "modules", "module")

	if !reflect.DeepEqual(got, []string{"smart", "raid"}) {
		t.Fatalf("ListOption = %#v", got)
	}
	got[0] = "changed"
	if source[0] != "smart" {
		t.Fatal("ListOption returned source slice without copying")
	}
}

func TestListOptionReturnsEmptyWhenMissingOrLoadFails(t *testing.T) {
	originalLoadConfig := loadConfig
	originalGet := get
	defer func() {
		loadConfig = originalLoadConfig
		get = originalGet
	}()

	loadConfig = func(config string, force bool) error {
		return nil
	}
	get = func(config string, section string, option string) ([]string, bool) {
		return nil, false
	}
	if got := ListOption("quickstart", "modules", "module"); got == nil || len(got) != 0 {
		t.Fatalf("ListOption missing = %#v, want empty slice", got)
	}

	loadConfig = func(config string, force bool) error {
		return errors.New("load failed")
	}
	get = func(config string, section string, option string) ([]string, bool) {
		return []string{"stale"}, true
	}
	if got := ListOption("quickstart", "modules", "module"); got == nil || len(got) != 0 {
		t.Fatalf("ListOption load failure = %#v, want empty slice", got)
	}
}
