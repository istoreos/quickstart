package service

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

func TestDeviceClassifierContract(t *testing.T) {
	t.Parallel()
	classifier := NewDeviceClassifier()
	tests := []struct {
		name       string
		input      DeviceClassificationInput
		brand      string
		category   string
		source     string
		confidence string
	}{
		{name: "fallback", input: DeviceClassificationInput{}, category: "computer", source: "fallback", confidence: "low"},
		{name: "ASUS manufacturer", input: DeviceClassificationInput{Manufacturer: "ASUSTek COMPUTER INC."}, brand: "ASUS", category: "computer", source: "manufacturer_default", confidence: "medium"},
		{name: "ASUS router model", input: DeviceClassificationInput{DisplayName: "ASUS RT-AX88U", Manufacturer: "ASUSTek COMPUTER INC."}, brand: "ASUS", category: "network", source: "model", confidence: "high"},
		{name: "Nintendo Switch", input: DeviceClassificationInput{Hostname: "Nintendo-Switch"}, category: "gaming", source: "model", confidence: "high"},
		{name: "network switch", input: DeviceClassificationInput{Hostname: "office-switch"}, category: "network", source: "hostname", confidence: "medium"},
		{name: "camera", input: DeviceClassificationInput{Hostname: "front-door-camera"}, category: "camera", source: "hostname", confidence: "medium"},
		{name: "NAS", input: DeviceClassificationInput{Hostname: "family-nas-server"}, category: "storage", source: "hostname", confidence: "medium"},
		{name: "printer", input: DeviceClassificationInput{Hostname: "office-printer"}, category: "printer", source: "hostname", confidence: "medium"},
		{name: "watch", input: DeviceClassificationInput{DisplayName: "Apple Watch", Manufacturer: "Apple, Inc."}, brand: "Apple", category: "wearable", source: "model", confidence: "high"},
		{name: "randomized MAC no manufacturer", input: DeviceClassificationInput{Hostname: "client"}, category: "computer", source: "fallback", confidence: "low"},
		{name: "component vendor is not device brand", input: DeviceClassificationInput{Manufacturer: "Intel Corporate"}, category: "computer", source: "fallback", confidence: "low"},
		{name: "realtek is not device brand", input: DeviceClassificationInput{Manufacturer: "Realtek Semiconductor Corp."}, category: "computer", source: "fallback", confidence: "low"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := classifier.Classify(test.input)
			if got.Brand != test.brand || got.Category != test.category || got.Source != test.source || got.Confidence != test.confidence {
				t.Fatalf("classification = %#v", got)
			}
			if got.Manufacturer != test.input.Manufacturer {
				t.Fatalf("manufacturer = %q, want raw %q", got.Manufacturer, test.input.Manufacturer)
			}
		})
	}
}

func TestDeviceClassifierIsDeterministic(t *testing.T) {
	t.Parallel()
	input := DeviceClassificationInput{Hostname: "living-room-tv", Manufacturer: "Samsung Electronics"}
	first := NewDeviceClassifier().Classify(input)
	second := NewDeviceClassifier().Classify(input)
	if *first != *second {
		t.Fatalf("classification changed: %#v != %#v", first, second)
	}
}

type classificationFixture struct {
	Name     string                    `json:"name"`
	Group    string                    `json:"group"`
	Input    DeviceClassificationInput `json:"input"`
	Expected struct {
		Brand      string `json:"brand"`
		Category   string `json:"category"`
		Source     string `json:"source"`
		Confidence string `json:"confidence"`
	} `json:"expected"`
}

func TestDeviceClassifierReviewedFixtureLibrary(t *testing.T) {
	data, err := os.ReadFile("testdata/device_classification_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []classificationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 60 {
		t.Fatalf("fixtures = %d, want at least 60", len(fixtures))
	}

	groups := map[string]int{}
	manufacturers := map[string]bool{}
	brands := map[string]bool{}
	sources := map[string]int{}
	seenNames := map[string]bool{}
	classifier := NewDeviceClassifier()
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			if fixture.Name == "" || seenNames[fixture.Name] {
				t.Fatalf("fixture name is empty or duplicated: %q", fixture.Name)
			}
			seenNames[fixture.Name] = true
			got := classifier.Classify(fixture.Input)
			if got.Brand != fixture.Expected.Brand || got.Category != fixture.Expected.Category || got.Source != fixture.Expected.Source || got.Confidence != fixture.Expected.Confidence {
				t.Fatalf("classification = %#v, expected = %#v", got, fixture.Expected)
			}
			if got.Manufacturer != fixture.Input.Manufacturer {
				t.Fatalf("raw manufacturer changed from %q to %q", fixture.Input.Manufacturer, got.Manufacturer)
			}
		})
		groups[fixture.Group]++
		sources[fixture.Expected.Source]++
		if fixture.Input.Manufacturer != "" {
			manufacturers[fixture.Input.Manufacturer] = true
		}
		if fixture.Expected.Brand != "" {
			brands[fixture.Expected.Brand] = true
		}
	}
	if len(groups) != 12 {
		t.Fatalf("fixture groups = %#v, want 12 category/fallback groups", groups)
	}
	for group, count := range groups {
		if count < 5 {
			t.Fatalf("group %s has %d fixtures, want at least 5", group, count)
		}
	}
	if len(manufacturers) < 20 || len(brands) < 10 {
		t.Fatalf("coverage manufacturers=%d brands=%d", len(manufacturers), len(brands))
	}
	groupNames := make([]string, 0, len(groups))
	for group := range groups {
		groupNames = append(groupNames, group)
	}
	sort.Strings(groupNames)
	t.Logf("classification diagnostics: fixtures=%d groups=%v manufacturers=%d brands=%d sources=%v fallback=%d", len(fixtures), groupNames, len(manufacturers), len(brands), sources, sources["fallback"])
}

func TestReviewedDeviceBrandRegistryHasAuditNotesAndRejectsNearMatches(t *testing.T) {
	for _, entry := range reviewedDeviceBrands {
		if entry.brand == "" || entry.reviewNote == "" {
			t.Fatalf("brand registry entry is missing audit data: %#v", entry)
		}
	}
	for _, manufacturer := range []string{"Intel Corporate", "Realtek Semiconductor Corp.", "ASUSTeK Router Components", "Appleton Systems", "Samsungton Labs", "Microsoft Wi-Fi Direct Virtual Adapter"} {
		if got := normalizedDeviceBrand(manufacturer); got != "" {
			t.Fatalf("near/component manufacturer %q matched brand %q", manufacturer, got)
		}
	}
}
