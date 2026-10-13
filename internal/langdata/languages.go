// Package langdata supplies the bundled ISO language registry to metadata consumers.
package langdata

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed registry.json
var files embed.FS
var Revision string
var aliases map[string]string

func init() {
	raw, err := files.ReadFile("registry.json")
	if err != nil {
		panic(err)
	}
	Revision = fmt.Sprintf("%x", sha256.Sum256(raw))
	var rows [][]json.RawMessage
	if err = json.Unmarshal(raw, &rows); err != nil {
		panic(err)
	}
	aliases = make(map[string]string, len(rows)*2)
	for _, row := range rows {
		var canonical string
		var codes, names []string
		json.Unmarshal(row[3], &canonical)
		json.Unmarshal(row[4], &codes)
		json.Unmarshal(row[5], &names)
		for _, value := range append(codes, names...) {
			key := strings.ToLower(strings.TrimSpace(value))
			if key != "" {
				if _, ok := aliases[key]; !ok {
					aliases[key] = canonical
				}
			}
		}
	}
	for _, row := range rows {
		var canonical string
		var codes []string
		json.Unmarshal(row[3], &canonical)
		json.Unmarshal(row[4], &codes)
		for _, code := range codes {
			aliases[code] = canonical
		}
	}

}

var regional = regexp.MustCompile(`^([a-z]{2,3})[-_](?:[a-z]{2}|[0-9]{3}|[a-z]{4})(?:[-_][a-z0-9]{5,8})*$`)

func IsUnknown(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "__unknown__", "und", "unknown", "unknown language", "undetermined", "undefined", "n/a", "unbekannt", "desconocido", "sconosciuto":
		return true
	}
	return false
}
func Canonical(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	if found := aliases[key]; found != "" {
		return found
	}
	// Resolve regional/script metadata identifiers through a registered language code.
	if match := regional.FindStringSubmatch(key); len(match) > 0 {
		return aliases[match[1]]
	}

	return ""
}
