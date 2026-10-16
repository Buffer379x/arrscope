package app

import "fmt"

func defaultListInfoOrder() []string {
	return []string{"link", "media", "details", "metadata", "recognition"}
}

// Upgrade old layouts once, preserving the relative order and visibility of other columns.
func normalizeListInfo(cfg *Settings) error {
	legacy := cfg.ListInfoOrder == nil
	if legacy {
		cfg.ListInfoOrder = defaultListInfoOrder()
	}
	if cfg.ListInfoVisible == nil {
		cfg.ListInfoVisible = defaultListInfoOrder()
	}
	// Add the new action once to valid pre-0.9.5 layouts. Preserve explicit hidden choices.
	if len(cfg.ListInfoOrder) == 4 && !containsColumn(cfg.ListInfoOrder, "media") {
		seen := map[string]bool{}
		for _, key := range cfg.ListInfoOrder {
			if key != "link" && key != "details" && key != "metadata" && key != "recognition" || seen[key] {
				return fmt.Errorf("Invalid Info action selection or order")
			}
			seen[key] = true
		}
		order := []string{}
		for _, key := range cfg.ListInfoOrder {
			order = append(order, key)
			if key == "link" {
				order = append(order, "media")
			}
		}
		cfg.ListInfoOrder = order
		if containsColumn(cfg.ListInfoVisible, "link") && !containsColumn(cfg.ListInfoVisible, "media") {
			cfg.ListInfoVisible = append(cfg.ListInfoVisible, "media")
		}
	}
	allowed := map[string]bool{"link": true, "media": true, "details": true, "metadata": true, "recognition": true}
	if len(cfg.ListInfoOrder) != len(allowed) {
		return fmt.Errorf("Info order must include all actions")
	}
	for _, values := range [][]string{cfg.ListInfoOrder, cfg.ListInfoVisible} {
		seen := map[string]bool{}
		for _, value := range values {
			if !allowed[value] || seen[value] {
				return fmt.Errorf("Invalid Info action selection or order")
			}
			seen[value] = true
		}
	}
	for service, order := range cfg.ColumnOrder {
		if !containsColumn(order, "info") {
			cfg.ColumnOrder[service] = append([]string{"info"}, order...)
		}
	}
	if legacy {
		for service, visible := range cfg.VisibleColumns {
			if !containsColumn(visible, "info") {
				cfg.VisibleColumns[service] = append([]string{"info"}, visible...)
			}
		}
	}
	return nil
}
