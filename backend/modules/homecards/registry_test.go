package homecards

import (
	"encoding/json"
	"fmt"
	"github.com/istoreos/quickstart/backend/modules/homeconfig"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample(t *testing.T) Manifest {
	t.Helper()
	raw, e := os.ReadFile("../../../docs/contracts/fastnet-card.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	json.Unmarshal(raw, &f)
	m, e := Parse(f.Manifest)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func put(t *testing.T, root, name string, m Manifest) {
	t.Helper()
	raw, _ := json.Marshal(m)
	if e := os.WriteFile(filepath.Join(root, name), raw, 0600); e != nil {
		t.Fatal(e)
	}
}
func TestDiscoveryBoundsAndDuplicates(t *testing.T) {
	root := t.TempDir()
	if len(Discover(root).Cards) != 0 {
		t.Fatal("empty")
	}
	m := sample(t)
	put(t, root, "fastnet.json", m)
	if len(Discover(root).Cards) != 1 {
		t.Fatal("one")
	}
	bad := m
	bad.ABI = "future"
	put(t, root, "duplicate.json", bad)
	if len(Discover(root).Cards) != 0 {
		t.Fatal("duplicate invalid ABI must not win")
	}
	os.Remove(filepath.Join(root, "duplicate.json"))
	os.Symlink(filepath.Join(root, "fastnet.json"), filepath.Join(root, "linked.json"))
	os.WriteFile(filepath.Join(root, "huge.json"), []byte(strings.Repeat("x", MaxFile+1)), 0600)
	os.WriteFile(filepath.Join(root, "broken.json"), []byte("{"), 0600)
	got := Discover(root)
	if len(got.Cards) != 1 || len(got.Issues) != 3 {
		t.Fatalf("invalid files: %+v", got)
	}
	root = t.TempDir()
	for i := 0; i < 32; i++ {
		m.ID = fmt.Sprintf("plugin.card%d", i)
		put(t, root, fmt.Sprintf("card%d.json", i), m)
	}
	if len(Discover(root).Cards) != 32 {
		t.Fatal("32 entries")
	}
	put(t, root, "extra.json", m)
	if len(Discover(root).Cards) != 0 {
		t.Fatal("33 entries accepted")
	}
}
func TestParseRejectsExecutableOrUnknownContract(t *testing.T) {
	m := sample(t)
	for _, entry := range []string{"https://other/card.12345678.js", "/luci-static/f/../card.12345678.js", "/luci-static/f/card.12345678.js?x=1", "/luci-static/f/card.js"} {
		bad := m
		bad.Entry = entry
		raw, _ := json.Marshal(bad)
		if _, e := Parse(raw); e == nil {
			t.Fatal(entry)
		}
	}
	raw, _ := json.Marshal(m)
	for _, bad := range []string{strings.Replace(string(raw), `"schemaVersion":1`, `"schemaVersion":2`, 1), strings.Replace(string(raw), `"schemaVersion":1`, `"schemaVersion":1,"schemaVersion":1`, 1), strings.Replace(string(raw), `"entry":`, `"unknown":`, 1)} {
		if _, e := Parse([]byte(bad)); e == nil {
			t.Fatal("invalid accepted")
		}
	}
}
func TestRegistryHTTPRequiresAuthentication(t *testing.T) {
	root := t.TempDir()
	put(t, root, "fastnet.json", sample(t))
	deny := httptest.NewRecorder()
	Handler(root, nil).ServeHTTP(deny, httptest.NewRequest("GET", "/", nil))
	if deny.Code != 403 {
		t.Fatal(deny.Code)
	}
	allow := func(*http.Request) (homeconfig.Access, error) { return homeconfig.Access{Read: true}, nil }
	server := httptest.NewServer(Handler(root, allow))
	defer server.Close()
	res, e := http.Get(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	var r Registry
	json.NewDecoder(res.Body).Decode(&r)
	if res.StatusCode != 200 || len(r.Cards) != 1 || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("HTTP registry")
	}
}

func TestIncompatibleRegistrationRetainsSafeIdentity(t *testing.T) {
	for _, abi := range []string{"quickstart-card-v2", "unknown"} {
		root := t.TempDir()
		m := sample(t)
		m.ABI = abi
		put(t, root, "fastnet.json", m)
		got := Discover(root)
		if len(got.Cards) != 0 || len(got.Issues) != 1 || got.Issues[0].ID != "plugin.fastnet" || got.Issues[0].Code != "unsupported_abi" {
			t.Fatalf("%+v", got)
		}
	}
}
