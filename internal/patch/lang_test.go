package patch

import "testing"

func TestLanguageFor(t *testing.T) {
	for in, want := range map[string]string{
		"french": "fr", "fr": "fr", "fr-FR": "fr", "fr_CA": "fr", "de-DE": "de", "pt-BR": "pt",
		"zh-CN": "zh-hans", "zh-Hans": "zh-hans", "zh-TW": "zh-hant", "zh-Hant-HK": "zh-hant",
		"ja-JP": "ja", "ko-KR": "ko", "nl-NL": "en", "": "en", "schinese": "zh-hans",
	} {
		if got := LanguageFor(in); got != want {
			t.Errorf("LanguageFor(%q) = %q, want %q", in, got, want)
		}
	}
}
