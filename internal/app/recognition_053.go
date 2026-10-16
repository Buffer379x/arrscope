package app

import (
	"context"
	"strings"
)

func ruleIsTag(r RecognitionRule) bool { return r.Tag == nil || *r.Tag }
func ruleFieldName(f string) string {
	if n := map[string]string{"resolution": "Resolution", "codec": "Codec", "source": "Source", "processing": "Processing", "hdr": "HDR", "bitDepth": "Bit depth", "audio": "Audio", "channels": "Channels", "language": "Language", "custom": "Custom"}[f]; n != "" {
		return n
	}
	return f
}
func unknownTagLabel(f string) string {
	if f == "audio" {
		f = "audio-codec"
	}
	if f == "bitDepth" {
		f = "bit-depth"
	}
	return f + "-unknown"
}
func uhdBluRayDefaultRule() RecognitionRule {
	return RecognitionRule{ID: "default-source-uhd-bluray", Name: "Source / UHD", Field: "source", Match: "regex", Pattern: `(?:^|[ ._\[\]()-])UHD(?:$|[ ._\[\]()-])`, Value: "UHD", Mode: "override", Priority: 94, Enabled: true, Unless: `(?:^|[ ._\[\]()-])(?:WEB[ ._-]?(?:DL|Rip)|HDTV|DVD)(?:$|[ ._\[\]()-])`}
}
func upgradeRules053(rules []RecognitionRule) ([]RecognitionRule, bool) {
	next := append([]RecognitionRule{}, rules...)
	changed := false
	found := false
	names := map[string]string{}
	for _, r := range DefaultRecognitionRules() {
		names[r.ID] = r.Field + " · " + strings.TrimPrefix(r.Name, ruleFieldName(r.Field)+" / ")
	}
	for i, r := range next {
		if old, ok := names[r.ID]; ok && r.Name == old {
			next[i].Name = ruleFieldName(r.Field) + " / " + strings.TrimPrefix(r.Name, r.Field+" · ")
			changed = true
		}

		if r.ID == "default-source-uhd-bluray" {
			found = true
		}
		if r.Field == "source" && strings.EqualFold(r.Value, "UHD") {
			probe := r
			probe.Enabled = true
			if e, err := CompileRecognition([]RecognitionRule{probe}); err == nil && e.Recognize("Example.UHD.BluRay.mkv").Values["source"] == "UHD" {
				found = true
			}
		}
	}
	if !found && len(next) < 256 {
		next = append(next, uhdBluRayDefaultRule())
		changed = true
	}
	return next, changed
}

// Presentation registry is separate from recognized data and outbound categories.
func (s *Store) DisplayTags(ctx context.Context) (map[string][]string, error) {
	engine := s.Recognition()
	out := map[string][]string{}
	sets := map[string]map[string]bool{}
	informationFields := map[string]bool{}
	for _, r := range engine.Rules {
		if r.Enabled && !ruleIsTag(r) {
			informationFields[r.Field] = true
		}
	}
	for _, r := range engine.Rules {
		if !r.Enabled || !ruleIsTag(r) {
			continue
		}
		if sets[r.Field] == nil {
			sets[r.Field] = map[string]bool{}
		}
		if !strings.Contains(r.Value, "$") {
			value := normalizeRuleValue(r.Field, r.Value)
			sets[r.Field][value] = true
		}
		if r.Field == "codec" {
			sets["family"] = map[string]bool{"H.264/AVC": true, "H.265/HEVC": true, "AV1": true, "VP9": true, "MPEG-2": true}
		}
	}
	for _, f := range recognitionFields {
		if sets[f] == nil {
			sets[f] = map[string]bool{}
		}
		if f != "custom" {
			sets[f]["__unknown__"] = true
		}
	}
	// API-only codecs also have a normal tag presentation. Per-file information
	// rules and the informationFields check below still control plain values.
	codecRows, codecErr := s.DB.QueryContext(ctx, "SELECT DISTINCT codec FROM files WHERE codec<>''")
	if codecErr != nil {
		return nil, codecErr
	}
	for codecRows.Next() {
		var value string
		if err := codecRows.Scan(&value); err != nil {
			codecRows.Close()
			return nil, err
		}
		sets["codec"][value] = true
	}
	codecErr = codecRows.Err()
	codecRows.Close()
	if codecErr != nil {
		return nil, codecErr
	}
	// Dynamic captures use only filename evidence from enabled, tag-marked rules.
	for _, r := range engine.Rules {
		if !r.Enabled || !ruleIsTag(r) || !strings.Contains(r.Value, "$") {
			continue
		}
		field := r.Field
		if field == "custom" {
			continue
		}
		rows, e := s.DB.QueryContext(ctx, "SELECT DISTINCT "+field+" FROM files f WHERE EXISTS(SELECT 1 FROM json_each(f.data,'$.evidence.'||?||'.rules') x WHERE x.value=?)", field, r.ID)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var v string
			if e = rows.Scan(&v); e != nil {
				rows.Close()
				return nil, e
			}
			if v != "" {
				sets[field][v] = true
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	}
	// Manually assigned custom tags remain tags independently of recognition rules.
	rows, e := s.DB.QueryContext(ctx, "SELECT DISTINCT c.tag_id FROM file_custom c WHERE NOT EXISTS(SELECT 1 FROM file_custom_auto a WHERE a.file_rowid=c.file_rowid AND a.tag_id=c.tag_id) OR EXISTS(SELECT 1 FROM custom_overrides o JOIN files f ON f.rowid=c.file_rowid WHERE o.service=f.service AND o.file_id=f.id AND o.tag_id=c.tag_id AND o.state=1)")
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		sets["custom"][id] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for f, values := range sets {
		out[f] = []string{}
		for v := range values {
			if informationFields[f] && f != "custom" && v != "__unknown__" {
				sep := " + "
				if f == "language" {
					sep = ", "
				}
				var exists bool
				if e = s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM files f WHERE (f."+f+"=? OR instr(?||f."+f+"||?,?||?||?)>0) AND NOT EXISTS(SELECT 1 FROM json_each(f.data,'$.informationOnly.'||?) i WHERE i.value=?))", v, sep, sep, sep, v, sep, f, v).Scan(&exists); e != nil {
					return nil, e
				}
				if !exists {
					continue
				}
			}
			out[f] = append(out[f], v)
		}
	}
	return out, nil
}

// Only the rules that contributed the final filename result determine whether
// that value is informational. A different rule for the same value must not
// turn a disabled Tag checkbox back on for this file.
func (e *RecognitionEngine) informationOnly(filename string, result RecognitionResult) map[string][]string {
	if !e.hasInformationRules {
		return nil
	}
	out := map[string][]string{}
	for field, ids := range result.Matches {
		informational := false
		for _, id := range ids {
			if !ruleIsTag(e.ruleCatalog[id].RecognitionRule) {
				informational = true
			}
		}
		if !informational {
			continue
		}
		tags := map[string]bool{}
		for _, id := range ids {
			r := e.ruleCatalog[id]
			if !ruleIsTag(r.RecognitionRule) {
				continue
			}
			for _, m := range r.re.FindAllStringSubmatchIndex(filename, -1) {
				value := r.Value
				if r.Match == "regex" {
					value = string(r.re.ExpandString(nil, r.Value, filename, m))
				}
				tags[normalizeRuleValue(field, value)] = true
			}
		}
		sep := " + "
		if field == "language" {
			sep = ", "
		}
		for _, v := range strings.Split(result.Values[field], sep) {
			if v != "" && !tags[v] {
				out[field] = append(out[field], v)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
