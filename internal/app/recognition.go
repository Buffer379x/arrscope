package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Rules affect filenames only. API normalization and Emby track attribution remain separate.
type RecognitionRule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Field    string `json:"field"`
	Match    string `json:"match"`
	Pattern  string `json:"pattern"`
	Value    string `json:"value"`
	Mode     string `json:"mode"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
	Consume  bool   `json:"consume"`
	Unless   string `json:"unless,omitempty"`
}
type RecognitionConfig struct {
	Rules    []RecognitionRule `json:"rules"`
	Revision string            `json:"revision"`
}
type RecognitionConflict struct {
	Field     string   `json:"field"`
	Values    []string `json:"values"`
	Rules     []string `json:"rules"`
	RuleNames []string `json:"ruleNames"`
}
type compiledRule struct {
	RecognitionRule
	re, unless *regexp.Regexp
}
type RecognitionEngine struct {
	config RecognitionConfig
	groups map[string][]compiledRule
	names  map[string]string
}
type RecognitionResult struct {
	Values    map[string]string     `json:"values"`
	Matches   map[string][]string   `json:"matches"`
	Origins   map[string]string     `json:"origins"`
	Conflicts []RecognitionConflict `json:"conflicts"`
}

var recognitionFields = []string{"resolution", "codec", "source", "processing", "hdr", "bitDepth", "audio", "channels", "language"}
var ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

func CompileRecognition(rules []RecognitionRule) (*RecognitionEngine, error) {
	if len(rules) > 256 {
		return nil, fmt.Errorf("At most 256 recognition rules are allowed")
	}
	rules = append([]RecognitionRule{}, rules...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority < rules[j].Priority
		}
		return rules[i].ID < rules[j].ID
	})
	engine := &RecognitionEngine{config: RecognitionConfig{Rules: rules}, groups: map[string][]compiledRule{}, names: map[string]string{}}
	ids := map[string]bool{}
	for _, r := range rules {
		validField := false
		for _, f := range recognitionFields {
			validField = validField || f == r.Field
		}
		if !ruleIDPattern.MatchString(r.ID) || ids[r.ID] || !validField || strings.TrimSpace(r.Name) == "" || len(r.Name) > 80 || r.Pattern == "" || len(r.Pattern) > 500 || strings.TrimSpace(r.Value) == "" || len(r.Value) > 80 || len(r.Unless) > 500 || r.Priority < 0 || r.Priority > 10000 {
			return nil, fmt.Errorf("Invalid recognition rule")
		}
		ids[r.ID] = true
		if r.Mode != "direct" && r.Mode != "fallback" && r.Mode != "override" {
			return nil, fmt.Errorf("Invalid recognition rule mode")
		}
		pattern := r.Pattern
		switch r.Match {
		case "token":
			pattern = `(?:^|[ ._\[\]()/,;|+-])` + regexp.QuoteMeta(pattern) + `(?:$|[ ._\[\]()/,;|+-])`
		case "regex":
		default:
			return nil, fmt.Errorf("Invalid recognition match type")
		}
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil || re.MatchString("") {
			return nil, fmt.Errorf("Invalid or empty-matching regular expression in rule %s", r.Name)
		}
		var unless *regexp.Regexp
		if r.Unless != "" {
			unless, err = regexp.Compile("(?i)" + r.Unless)
			if err != nil {
				return nil, fmt.Errorf("Invalid exclusion expression in rule %s", r.Name)
			}
		}
		compiled := compiledRule{r, re, unless}
		engine.names[r.ID] = r.Name
		if r.Enabled {
			engine.groups[r.Field+":"+r.Mode] = append(engine.groups[r.Field+":"+r.Mode], compiled)
		}
	}
	raw, _ := json.Marshal(rules)
	sum := sha256.Sum256(raw)
	engine.config.Revision = hex.EncodeToString(sum[:])
	return engine, nil
}
func (e *RecognitionEngine) Config() RecognitionConfig {
	return RecognitionConfig{Rules: append([]RecognitionRule{}, e.config.Rules...), Revision: e.config.Revision}
}
func normalizeRuleValue(field, value string) string {
	value = strings.TrimSpace(value)
	switch field {
	case "resolution":
		return normResolution(value)
	case "codec":
		return normCodec(value)
	case "language":
		return normLanguage(value)
	}
	return value
}
func (e *RecognitionEngine) Recognize(filename string) RecognitionResult {
	result := RecognitionResult{Values: map[string]string{}, Matches: map[string][]string{}, Origins: map[string]string{}, Conflicts: []RecognitionConflict{}}
	for _, field := range recognitionFields {
		compound := field == "hdr" || field == "audio" || field == "language"
		for _, mode := range []string{"direct", "fallback", "override"} {
			if mode == "fallback" && result.Values[field] != "" {
				continue
			}
			group := e.groups[field+":"+mode]
			if len(group) == 0 {
				continue
			}
			input := filename
			if field == "audio" {
				input = strings.NewReplacer("/", " ", ",", " ", ";", " ", "|", " ").Replace(input)
			}
			values, ids := []string{}, []string{}
			seen := map[string]bool{}
			for _, r := range group {
				if r.unless != nil && r.unless.MatchString(filename) {
					continue
				}
				matches := r.re.FindAllStringSubmatchIndex(input, -1)
				matched := false
				for _, match := range matches {
					if match[0] == match[1] {
						continue
					}
					matched = true
					value := r.Value
					if r.Match == "regex" {
						value = string(r.re.ExpandString(nil, r.Value, input, match))
					}
					value = normalizeRuleValue(field, value)
					if value != "" && !seen[value] {
						values = append(values, value)
						seen[value] = true
					}
				}
				if matched {
					ids = append(ids, r.ID)
					if r.Consume {
						for i := 0; i < 16 && r.re.MatchString(input); i++ {
							next := r.re.ReplaceAllString(input, " ")
							if len(next) >= len(input) {
								break
							}
							input = next
						}
					}
				}
			}
			if len(values) == 0 {
				continue
			}
			value := values[0]
			if compound {
				sep := " + "
				if field == "language" {
					sep = ", "
					sort.Strings(values)
				}
				value = strings.Join(values, sep)
			}
			result.Values[field] = value
			result.Matches[field] = ids
			result.Origins[field] = "filename"
			if mode != "direct" {
				result.Origins[field] = "rule"
			}
			// Direct and fallback conflicts are ignored only when an explicit override resolves them.
			kept := result.Conflicts[:0]
			for _, c := range result.Conflicts {
				if c.Field != field {
					kept = append(kept, c)
				}
			}
			result.Conflicts = kept
			compatible := field == "codec"
			for _, v := range values {
				compatible = compatible && family(v) == family(values[0])
			}
			if !compound && !compatible && len(values) > 1 {
				names := []string{}
				for _, id := range ids {
					names = append(names, e.names[id])
				}
				result.Conflicts = append(result.Conflicts, RecognitionConflict{field, values, ids, names})
			}
		}
	}
	return result
}

func DefaultRecognitionRules() []RecognitionRule {
	rules := []RecognitionRule{}
	add := func(field, pattern, value string, priority int) {
		rules = append(rules, RecognitionRule{ID: fmt.Sprintf("default-%s-%d", field, len(rules)), Name: field + " · " + value, Field: field, Match: "regex", Pattern: pattern, Value: value, Mode: "direct", Priority: priority, Enabled: true})
	}
	for _, v := range []string{"480p", "480i", "576p", "576i", "720p", "1080p", "1080i", "1440p", "2160p", "4320p", "4k", "8k"} {
		add("resolution", token(regexp.QuoteMeta(v)).String(), normResolution(v), 100)
	}
	for _, v := range []struct{ pattern, value string }{{`x[ ._-]?264`, "x264"}, {`x[ ._-]?265`, "x265"}, {`h[ ._-]?264|avc`, "AVC"}, {`h[ ._-]?265|hevc`, "HEVC"}, {`av1`, "AV1"}, {`xvid`, "XviD"}, {`divx`, "DivX"}, {`mpeg[ ._-]?2`, "MPEG-2"}, {`vc[ ._-]?1`, "VC-1"}} {
		add("codec", token(v.pattern).String(), v.value, 100)
	}
	for i, m := range sourceMatchers {
		add("source", m.re.String(), m.label, 100+i)
	}
	add("processing", remuxRE.String(), "Remux", 100)
	for _, v := range []struct {
		re    *regexp.Regexp
		value string
	}{{dolbyRE, "Dolby Vision"}, {hdr10PlusRE, "HDR10+"}, {hdr10RE, "HDR10"}, {hlgRE, "HLG"}, {hdrRE, "HDR"}} {
		add("hdr", v.re.String(), v.value, 100)
		if v.value == "HDR" {
			rules[len(rules)-1].Unless = token(`hdr10(?:\+|plus)?`).String()
		}
	}
	for _, v := range []string{"8", "10", "12"} {
		add("bitDepth", token(v+`[ ._-]?bit`).String(), v, 100)
	}
	for i, m := range audioMatchers {
		add("audio", m.re.String(), m.label, 100+i)
		rules[len(rules)-1].Consume = true
	}
	for _, v := range []string{"2.0", "2.1", "5.1", "7.1"} {
		add("channels", token(regexp.QuoteMeta(v)).String(), v, 100)
	}
	for _, m := range languageMatchers {
		add("language", m.re.String(), m.label, 100)
	}
	return rules
}

var defaultRecognitionEngine = func() *RecognitionEngine {
	e, err := CompileRecognition(DefaultRecognitionRules())
	if err != nil {
		panic(err)
	}
	return e
}()
