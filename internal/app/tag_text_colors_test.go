package app

import "testing"

func TestTagTextColorValidation(t *testing.T) {
	cfg := Settings{}
	if err := normalizeTagTextColors(&cfg); err != nil || cfg.TagTextColors == nil {
		t.Fatal("legacy settings must load", err)
	}
	for _, colors := range []map[string]string{
		{"codec:x264": "#000000", "source:__unknown__": "#ffffff"},
		{},
	} {
		cfg.TagTextColors = colors
		if err := normalizeTagTextColors(&cfg); err != nil {
			t.Fatal(err)
		}
	}
	for _, colors := range []map[string]string{
		{"codec:x264": "#123456"}, {"password:x": "#ffffff"}, {"codec:": "#ffffff"}, {"codec:x264": "#ffffff;display:none"},
	} {
		cfg.TagTextColors = colors
		if normalizeTagTextColors(&cfg) == nil {
			t.Fatal("invalid text color accepted", colors)
		}
	}
}
