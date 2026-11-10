// Package homecards discovers data-only manifests owned by installed packages.
package homecards

import (
	"encoding/json"
	"errors"
	"github.com/istoreos/quickstart/backend/modules/homeconfig"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const ABI = "quickstart-card-v1"
const MaxFile = 8192
const MaxResponse = 65536

type Size struct {
	MinWidth  int `json:"minWidth"`
	MinHeight int `json:"minHeight"`
}
type Manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Glyph         string          `json:"glyph"`
	ABI           string          `json:"abi"`
	Version       string          `json:"version"`
	Entry         string          `json:"entry"`
	APIBase       string          `json:"apiBase"`
	Forms         map[string]Size `json:"forms"`
	Stacked       []string        `json:"stacked"`
}
type Issue struct {
	Code string `json:"code"`
	ID   string `json:"id,omitempty"`
}
type Registry struct {
	SchemaVersion int        `json:"schemaVersion"`
	Cards         []Manifest `json:"cards"`
	Issues        []Issue    `json:"issues"`
}

var idPattern = regexp.MustCompile(`^plugin\.[a-z][a-z0-9-]{0,47}$`)
var entryPattern = regexp.MustCompile(`^/luci-static/[a-z][a-z0-9-]*/(?:[a-zA-Z0-9_-]+/)*[a-zA-Z0-9_-]+\.[a-f0-9]{8,64}\.js$`)
var apiPattern = regexp.MustCompile(`^/cgi-bin/luci/admin/services/[a-z][a-z0-9-]*/card$`)
var versionPattern = regexp.MustCompile(`^[A-Za-z0-9.+-]{1,64}$`)
var glyphPattern = regexp.MustCompile(`^[a-z-]{1,32}$`)
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}\.json$`)
var badTitle = regexp.MustCompile(`[\x00-\x1f\x7f]`)

func Parse(raw []byte) (Manifest, error) {
	var m Manifest
	if len(raw) > MaxFile || homeconfig.DecodeStrict(raw, &m) != nil {
		return m, errors.New("invalid_manifest")
	}
	if m.SchemaVersion > 1 {
		return m, errors.New("unsupported_protocol")
	}
	if m.SchemaVersion == 1 && m.ABI != "" && m.ABI != ABI {
		return m, errors.New("unsupported_abi")
	}
	if m.SchemaVersion != 1 || m.ABI != ABI || !idPattern.MatchString(m.ID) || strings.TrimSpace(m.Title) == "" || len([]rune(m.Title)) > 64 || badTitle.MatchString(m.Title) || !glyphPattern.MatchString(m.Glyph) || !versionPattern.MatchString(m.Version) || !entryPattern.MatchString(m.Entry) || !apiPattern.MatchString(m.APIBase) || len(m.Forms) < 1 || len(m.Forms) > 2 {
		return m, errors.New("invalid_manifest")
	}
	for form, size := range m.Forms {
		if (form != "standard" && form != "summary") || size.MinWidth < 240 || size.MinWidth > 1200 || size.MinHeight < 100 || size.MinHeight > 1600 {
			return m, errors.New("invalid_form")
		}
	}
	if len(m.Stacked) > 1 || len(m.Stacked) == 1 && (m.Stacked[0] != "summary" || m.Forms["summary"].MinWidth == 0) {
		return m, errors.New("invalid_stacked")
	}
	return m, nil
}
func Discover(root string) Registry {
	out := Registry{SchemaVersion: 1, Cards: []Manifest{}, Issues: []Issue{}}
	issue := func(code string, ids ...string) {
		item := Issue{Code: code}
		if len(ids) > 0 && idPattern.MatchString(ids[0]) {
			item.ID = ids[0]
		}
		out.Issues = append(out.Issues, item)
	}
	dir, err := os.Open(root)
	if errors.Is(err, os.ErrNotExist) {
		return out
	}
	if err != nil {
		issue("registry_unavailable")
		return out
	}
	defer dir.Close()
	names, err := dir.Readdirnames(33)
	if err != nil && err != io.EOF {
		issue("registry_unavailable")
		return out
	}
	if len(names) > 32 {
		issue("too_many_manifests")
		return out
	}
	sort.Strings(names)
	candidates := []Manifest{}
	counts := map[string]int{}
	for _, name := range names {
		if !namePattern.MatchString(name) {
			issue("invalid_filename")
			continue
		}
		path := filepath.Join(root, name)
		stat, err := os.Lstat(path)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() > MaxFile {
			issue("invalid_file")
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			issue("unreadable_manifest")
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(f, MaxFile+1))
		f.Close()
		if err != nil {
			issue("unreadable_manifest")
			continue
		}
		var identity struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &identity) == nil && identity.ID != "" {
			counts[identity.ID]++
		}
		m, err := Parse(raw)
		if err != nil {
			issue(err.Error(), identity.ID)
			continue
		}
		candidates = append(candidates, m)
	}
	for _, m := range candidates {
		if counts[m.ID] != 1 {
			issue("duplicate_id", m.ID)
			continue
		}
		out.Cards = append(out.Cards, m)
		raw, _ := json.Marshal(out)
		if len(raw) > MaxResponse-4096 {
			out.Cards = out.Cards[:len(out.Cards)-1]
			issue("registry_too_large")
			break
		}
	}
	return out
}
func Handler(root string, authorize homeconfig.Authorize) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, `{"error":"method"}`, 405)
			return
		}
		if authorize == nil {
			http.Error(w, `{"error":"forbidden"}`, 403)
			return
		}
		access, err := authorize(r)
		if err != nil || !access.Read {
			http.Error(w, `{"error":"forbidden"}`, 403)
			return
		}
		json.NewEncoder(w).Encode(Discover(root))
	})
}
