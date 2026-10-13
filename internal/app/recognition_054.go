package app

// Broaden only the untouched UHD BluRay pattern. Keep IDs, user names,
// priorities, enabled state and presentation flags; custom patterns stay intact.
func upgradeRules054(rules []RecognitionRule) ([]RecognitionRule, bool) {
	next := append([]RecognitionRule{}, rules...)
	changed := false
	const oldPattern = `(?:^|[ ._\[\]()-])UHD[ ._-]+BluRay(?:$|[ ._\[\]()-])`
	for i, r := range next {
		if r.ID == "default-source-uhd-bluray" && r.Field == "source" && r.Match == "regex" && r.Pattern == oldPattern && r.Value == "UHD" && r.Mode == "override" {
			next[i].Pattern = uhdBluRayDefaultRule().Pattern
			if r.Name == "Source / UHD BluRay" {
				next[i].Name = "Source / UHD"
			}
			changed = true
		}
	}
	return next, changed
}
