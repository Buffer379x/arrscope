package app

import "strings"

// Compatibility upgrades for stored 0.5.x filename rules. Keep their IDs and user customizations.
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

// Broaden only the untouched UHD BluRay pattern. Keep IDs, user names,
// priorities, enabled state and presentation flags; custom patterns stay intact.
func upgradeRules054(rules []RecognitionRule) ([]RecognitionRule, bool) {
	next := append([]RecognitionRule{}, rules...)
	changed := false
	const oldPattern = `(?:^|[ ._\[\]()-])UHD[ ._-]+BluRay(?:$|[ ._\[\]()-])`
	for i, r := range next {
		if r.ID == "default-source-uhd-bluray" && r.Field == "source" && r.Match == "regex" && r.Pattern == oldPattern && r.Value == "UHD" && r.Mode == "override" {
			next[i].Pattern = uhdBluRayDefaultRule().Pattern
			if r.Unless == "" {
				next[i].Unless = uhdBluRayDefaultRule().Unless
			}
			if r.Name == "Source / UHD BluRay" {
				next[i].Name = "Source / UHD"
			}
			changed = true
		}
	}
	return next, changed
}
