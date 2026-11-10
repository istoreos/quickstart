// Package homeconfig owns the bounded, code-free homepage configuration contract.
// Layout and launcher are independent domains; no business snapshot is stored here.
package homeconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 128 * 1024

var ErrInvalid = errors.New("invalid homepage configuration")
var ErrFuture = errors.New("newer homepage schema requires a newer version")

type Layout struct {
	SchemaVersion     int                   `json:"schemaVersion"`
	Rows              []Row                 `json:"rows"`
	WidgetPreferences map[string]Preference `json:"widgetPreferences"`
}
type Row struct {
	ID         string   `json:"id"`
	Template   string   `json:"template"`
	Height     float64  `json:"height"`
	HeightMode string   `json:"heightMode"`
	MinHeight  *float64 `json:"minHeight,omitempty"`
	Columns    []Column `json:"columns"`
}
type Column struct {
	Weight     float64   `json:"weight"`
	Cards      []*string `json:"cards"`
	Split      float64   `json:"split"`
	UpperShare *float64  `json:"upperShare,omitempty"`
}
type Preference struct {
	Mode        string `json:"mode"`
	IconRows    int    `json:"iconRows"`
	RuleVersion int    `json:"ruleVersion"`
}
type Launcher struct {
	SchemaVersion int      `json:"schemaVersion"`
	Favorites     []string `json:"favorites"`
	Links         []Link   `json:"links"`
}
type Link struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Glyph string `json:"glyph"`
}

var widgets = []string{"traffic", "connection", "networkInterface", "devices", "applications", "storage", "sharing", "docker", "downloads", "remote", "system"}
var templates = map[string][]int{"single": {1}, "pair": {1, 1}, "triple": {1, 1, 1}, "left-big": {1, 2}, "right-big": {2, 1}}
var summaries = map[string]bool{"connection": true, "networkInterface": true, "devices": true, "sharing": true, "docker": true, "downloads": true, "remote": true, "system": true}
var externalID = regexp.MustCompile(`^plugin\.[a-z][a-z0-9-]{0,47}$`)
var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+-]{0,127}$`)
var linkID = regexp.MustCompile(`^user-link:[A-Za-z0-9_-]{1,64}$`)
var icons = map[string]bool{"globe": true, "folder": true, "layers": true, "network": true, "shield": true, "download": true, "terminal": true, "grid": true}

// DecodeStrict also rejects duplicate object keys and excessive nesting. The
// normal encoding/json decoder alone would silently accept the last duplicate.
func DecodeStrict(raw []byte, into any) error {
	if len(raw) == 0 || len(raw) > MaxBytes || !utf8.Valid(raw) {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var visit func(int) error
	visit = func(depth int) error {
		if depth > 12 {
			return ErrInvalid
		}
		t, err := d.Token()
		if err != nil {
			return ErrInvalid
		}
		if delim, ok := t.(json.Delim); ok {
			if delim != '{' && delim != '[' {
				return ErrInvalid
			}
			keys := map[string]bool{}
			for d.More() {
				if delim == '{' {
					key, err := d.Token()
					if err != nil {
						return ErrInvalid
					}
					name, ok := key.(string)
					if !ok || keys[name] {
						return ErrInvalid
					}
					keys[name] = true
				}
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || (delim == '{' && end != json.Delim('}')) || (delim == '[' && end != json.Delim(']')) {
				return ErrInvalid
			}
		}
		return nil
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(into); err != nil {
		return ErrInvalid
	}
	return requiredFields(raw, reflect.TypeOf(into).Elem())
}

// Missing scalar fields and explicit nulls must not silently become Go zero
// values and then be saved as if the user had explicitly chosen them.
func requiredFields(raw []byte, kind reflect.Type) error {
	if kind == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		// null is allowed only as an empty widget slot (a pointer in an array).
		if kind.Kind() == reflect.Pointer {
			return nil
		}
		return ErrInvalid
	}
	if kind.Kind() == reflect.Pointer {
		return requiredFields(raw, kind.Elem())
	}
	switch kind.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			return ErrInvalid
		}
		for i := 0; i < kind.NumField(); i++ {
			field := kind.Field(i)
			tag := strings.Split(field.Tag.Get("json"), ",")
			name := tag[0]
			value, ok := fields[name]
			if !ok {
				if len(tag) < 2 || tag[1] != "omitempty" {
					return ErrInvalid
				}
				continue
			}
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return ErrInvalid
			}
			if err := requiredFields(value, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if json.Unmarshal(raw, &values) != nil {
			return ErrInvalid
		}
		for _, value := range values {
			if err := requiredFields(value, kind.Elem()); err != nil {
				return err
			}
		}
	case reflect.Map:
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &values) != nil {
			return ErrInvalid
		}
		for _, value := range values {
			if err := requiredFields(value, kind.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Canonical validates before producing bytes. Versions are checked before
// decoding known fields, so future documents remain intact and read-only.
func Canonical(domain string, raw []byte) ([]byte, error) {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if len(raw) > MaxBytes || json.Unmarshal(raw, &header) != nil {
		return nil, ErrInvalid
	}
	if domain == "layout" {
		if header.SchemaVersion > 4 {
			return nil, ErrFuture
		}
		var doc Layout
		if DecodeStrict(raw, &doc) != nil || validateLayout(doc) != nil {
			return nil, ErrInvalid
		}
		doc.SchemaVersion = 4
		return json.Marshal(doc)
	}
	if domain == "launcher" {
		if header.SchemaVersion > 1 {
			return nil, ErrFuture
		}
		var doc Launcher
		if DecodeStrict(raw, &doc) != nil || validateLauncher(doc) != nil {
			return nil, ErrInvalid
		}
		return json.Marshal(doc)
	}
	return nil, ErrInvalid
}

func finite(value, min, max float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= min && value <= max
}
func validateLayout(doc Layout) error {
	if (doc.SchemaVersion != 3 && doc.SchemaVersion != 4) || doc.Rows == nil || len(doc.Rows) > 32 || len(doc.WidgetPreferences) < len(widgets) || len(doc.WidgetPreferences) > len(widgets)+32 || doc.SchemaVersion == 3 && len(doc.WidgetPreferences) != len(widgets) {
		return ErrInvalid
	}
	for _, id := range widgets {
		p, ok := doc.WidgetPreferences[id]
		modeOK := p.Mode == "auto" || p.Mode == "standard" && id != "applications" || p.Mode == "summary" && summaries[id] || id == "applications" && (p.Mode == "grid" || p.Mode == "strip")
		if !ok || !modeOK || p.IconRows < 0 || p.IconRows > 3 || id != "applications" && p.IconRows != 0 || p.RuleVersion < 0 || p.RuleVersion > 2 {
			return ErrInvalid
		}
	}

	for id, p := range doc.WidgetPreferences {
		known := false
		for _, builtin := range widgets {
			if id == builtin {
				known = true
				break
			}
		}
		if known {
			continue
		}
		if doc.SchemaVersion != 4 || !externalID.MatchString(id) || (p.Mode != "auto" && p.Mode != "standard" && p.Mode != "summary") || p.IconRows != 0 || p.RuleVersion < 0 || p.RuleVersion > 2 {
			return ErrInvalid
		}
	}
	rows, used := map[string]bool{}, map[string]bool{}
	for _, row := range doc.Rows {
		shape, ok := templates[row.Template]
		if !ok || !safeID.MatchString(row.ID) || rows[row.ID] || len(row.Columns) != len(shape) || !finite(row.Height, 1, 1600) || (row.HeightMode != "content" && row.HeightMode != "chart") || row.MinHeight != nil && !finite(*row.MinHeight, 1, 1600) {
			return ErrInvalid
		}
		rows[row.ID] = true
		weight := 0.0
		for i, column := range row.Columns {
			if len(column.Cards) != shape[i] || !finite(column.Weight, 0.01, 100) || !finite(column.Split, 0.01, 99.99) || column.UpperShare != nil && (shape[i] != 2 || !finite(*column.UpperShare, 0.01, 99.99)) {
				return ErrInvalid
			}
			weight += column.Weight
			for _, id := range column.Cards {
				if id == nil {
					continue
				}
				p, ok := doc.WidgetPreferences[*id]
				if !ok || used[*id] {
					return ErrInvalid
				}
				used[*id] = true
				if shape[i] == 2 {
					mode := p.Mode
					if mode == "auto" {
						mode = "summary"
						if *id == "connection" && p.RuleVersion == 2 {
							mode = "standard"
						}
					}
					if !(*id == "connection" && mode == "standard") && !((summaries[*id] || externalID.MatchString(*id)) && *id != "devices" && mode == "summary") {
						return ErrInvalid
					}
				}
			}
		}
		if math.Abs(weight-100) > .01 {
			return ErrInvalid
		}
	}
	return nil
}

func validateLauncher(doc Launcher) error {
	if doc.SchemaVersion != 1 || doc.Favorites == nil || doc.Links == nil || len(doc.Favorites) > 24 || len(doc.Links) > 64 {
		return ErrInvalid
	}
	links := map[string]bool{}
	for _, link := range doc.Links {
		if !linkID.MatchString(link.ID) || links[link.ID] || link.Name != strings.TrimSpace(link.Name) || link.Name == "" || utf8.RuneCountInString(link.Name) > 64 || strings.IndexFunc(link.Name, unicode.IsControl) >= 0 || !icons[link.Glyph] {
			return ErrInvalid
		}
		if len(link.URL) > 2048 || strings.IndexFunc(link.URL, func(r rune) bool { return r <= 32 || r == 127 || r == '\\' }) >= 0 {
			return ErrInvalid
		}
		u, err := url.Parse(link.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
			return ErrInvalid
		}
		links[link.ID] = true
	}
	pins := map[string]bool{}
	for _, id := range doc.Favorites {
		if pins[id] || !(safeID.MatchString(id) || linkID.MatchString(id) && links[id]) {
			return ErrInvalid
		}
		pins[id] = true
	}
	return nil
}
