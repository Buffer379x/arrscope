package app

import "testing"

func TestUserUltraHDRuleSharesDefaultPriorityAndRespectsMode(t *testing.T) {
	for _, pattern := range []string{"Ultra.HD", ".Ultra.HD."} {
		for _, mode := range []string{"direct", "fallback", "override"} {
			rules := append(DefaultRecognitionRules(), customRule("user-ultra", "source", pattern, "UHD", mode, 95))
			engine, err := CompileRecognition(rules)
			if err != nil {
				t.Fatal(err)
			}
			got := engine.Recognize("Movie.Ultra.HD.BluRay.mkv")
			expected := "UHD"
			if mode == "fallback" {
				expected = "BluRay"
			}
			if got.Values["source"] != expected {
				t.Fatalf("%s %s: %#v", pattern, mode, got)
			}
			if mode == "override" && len(got.Conflicts) != 0 {
				t.Fatal("explicit override should resolve the direct source contradiction", got)
			}
			if mode == "direct" && len(got.Conflicts) == 0 {
				t.Fatal("different direct source results must retain their warning")
			}
		}
	}
}
func TestExplicitSeparatorTokensKeepLiteralInteriorAndBoundaries(t *testing.T) {
	for _, pattern := range []string{"Ultra.HD", ".Ultra.HD.", ".Ultra.HD", "Ultra.HD."} {
		rule := customRule("literal", "source", pattern, "UHD", "override", 95)
		engine, err := CompileRecognition([]RecognitionRule{rule})
		if err != nil {
			t.Fatal(err)
		}
		if engine.Recognize("Movie.Ultra.HD.BluRay.mkv").Values["source"] != "UHD" {
			t.Fatal(pattern)
		}
		for _, filename := range []string{"Movie.XUltra.HD.BluRay.mkv", "Movie.Ultra.HDX.BluRay.mkv", "Movie.Ultra-HD.BluRay.mkv"} {
			if engine.Recognize(filename).Values["source"] != "" {
				t.Fatalf("%s falsely matched %s", pattern, filename)
			}
		}
	}
	rule := customRule("later-user", "source", "Ultra.HD", "UHD", "direct", 500)
	engine, err := CompileRecognition(append(DefaultRecognitionRules(), rule))
	if err != nil {
		t.Fatal(err)
	}
	if engine.Recognize("Movie.Ultra.HD.BluRay.mkv").Values["source"] != "BluRay" {
		t.Fatal("priority rather than user/default origin must choose the first direct result")
	}
}
