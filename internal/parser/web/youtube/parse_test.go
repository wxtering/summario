package youtube_test

import (
	"testing"
	"tldr/internal/parser/web/youtube"
)

func TestSelectCaptionTrack(t *testing.T) {
	t.Run("empty tracks slice returns nil and false", func(t *testing.T) {
		track, ok := youtube.SelectCaptionTrack(nil, "ru")
		if ok || track != nil {
			t.Fatalf("expected nil and false, got ok=%v, track=%v", ok, track)
		}
	})

	t.Run("priority 1: prefers target language manual over asr", func(t *testing.T) {
		tracks := []youtube.CaptionTrack{
			{BaseURL: "url-ru-asr", LanguageCode: "ru", Kind: "asr"},
			{BaseURL: "url-ru-manual", LanguageCode: "ru", Kind: ""},
			{BaseURL: "url-en-manual", LanguageCode: "en", Kind: ""},
		}

		track, ok := youtube.SelectCaptionTrack(tracks, "ru")
		if !ok || track == nil {
			t.Fatalf("expected track, got ok=%v", ok)
		}
		if track.BaseURL != "url-ru-manual" {
			t.Errorf("expected url-ru-manual, got %s", track.BaseURL)
		}
	})

	t.Run("priority 2: uses target language asr if manual is unavailable", func(t *testing.T) {
		tracks := []youtube.CaptionTrack{
			{BaseURL: "url-en-manual", LanguageCode: "en", Kind: ""},
			{BaseURL: "url-ru-asr", LanguageCode: "ru", Kind: "asr"},
		}

		track, ok := youtube.SelectCaptionTrack(tracks, "ru")
		if !ok || track == nil {
			t.Fatalf("expected track, got ok=%v", ok)
		}
		if track.BaseURL != "url-ru-asr" {
			t.Errorf("expected url-ru-asr, got %s", track.BaseURL)
		}
	})

	t.Run("priority 3: falls back to english manual if target language is absent", func(t *testing.T) {
		tracks := []youtube.CaptionTrack{
			{BaseURL: "url-es-manual", LanguageCode: "es", Kind: ""},
			{BaseURL: "url-en-asr", LanguageCode: "en", Kind: "asr"},
			{BaseURL: "url-en-manual", LanguageCode: "en", Kind: ""},
		}

		track, ok := youtube.SelectCaptionTrack(tracks, "ru")
		if !ok || track == nil {
			t.Fatalf("expected track, got ok=%v", ok)
		}
		if track.BaseURL != "url-en-manual" {
			t.Errorf("expected url-en-manual, got %s", track.BaseURL)
		}
	})

	t.Run("priority 4: falls back to english asr if manual english is absent", func(t *testing.T) {
		tracks := []youtube.CaptionTrack{
			{BaseURL: "url-es-manual", LanguageCode: "es", Kind: ""},
			{BaseURL: "url-en-asr", LanguageCode: "en", Kind: "asr"},
		}

		track, ok := youtube.SelectCaptionTrack(tracks, "ru")
		if !ok || track == nil {
			t.Fatalf("expected track, got ok=%v", ok)
		}
		if track.BaseURL != "url-en-asr" {
			t.Errorf("expected url-en-asr, got %s", track.BaseURL)
		}
	})

	t.Run("priority 5: falls back to first track if target and english are absent", func(t *testing.T) {
		tracks := []youtube.CaptionTrack{
			{BaseURL: "url-de-asr", LanguageCode: "de", Kind: "asr"},
			{BaseURL: "url-fr-manual", LanguageCode: "fr", Kind: ""},
		}

		track, ok := youtube.SelectCaptionTrack(tracks, "ru")
		if !ok || track == nil {
			t.Fatalf("expected track, got ok=%v", ok)
		}
		if track.BaseURL != "url-de-asr" {
			t.Errorf("expected url-de-asr (first track), got %s", track.BaseURL)
		}
	})
}
