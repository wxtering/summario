package youtube

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"tldr/internal/models"
)

var (
	// Matches exact 11-char YouTube video ID (anchored strictly from start to end of string)
	videoIDRegex = regexp.MustCompile(`\A[a-zA-Z0-9_-]{11}\z`)
)

// ExtractVideoID parses a raw string or URL to obtain the 11-char YouTube video ID.
func ExtractVideoID(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", models.ErrInvalidURL
	}

	// 1. If source is already a plain 11-char ID
	if videoIDRegex.MatchString(source) {
		return source, nil
	}

	// 2. Handle scheme-less URLs (e.g. youtube.com/watch?v=ID)
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		source = "https://" + source
	}

	u, err := url.Parse(source)
	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrInvalidURL, err)
	}

	// Normalize hostname
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	host = strings.TrimPrefix(host, "music.")

	switch host {
	case "youtube.com", "youtube-nocookie.com":
		if u.Path == "/watch" {
			v := u.Query().Get("v")
			if videoIDRegex.MatchString(v) {
				return v, nil
			}
		}

		// Handle /embed/ID, /v/ID, /vi/ID, /live/ID, /shorts/ID
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			switch parts[0] {
			case "embed", "v", "vi", "live", "shorts":
				if videoIDRegex.MatchString(parts[1]) {
					return parts[1], nil
				}
			}
		}

	case "youtu.be":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 1 && videoIDRegex.MatchString(parts[0]) {
			return parts[0], nil
		}
	}

	return "", models.ErrInvalidURL
}

// InnerTube JSON Request Payload
type innerTubePayload struct {
	Context innerTubeContext `json:"context"`
	VideoID string           `json:"videoId"`
}

type innerTubeContext struct {
	Client innerTubeClient `json:"client"`
}

type innerTubeClient struct {
	ClientName        string `json:"clientName"`
	ClientVersion     string `json:"clientVersion"`
	AndroidSDKVersion int    `json:"androidSdkVersion"`
	OSName            string `json:"osName"`
	OSVersion         string `json:"osVersion"`
}

func newInnerTubePayload(videoID string) innerTubePayload {
	return innerTubePayload{
		Context: innerTubeContext{
			Client: innerTubeClient{
				ClientName:        clientName,
				ClientVersion:     clientVersion,
				AndroidSDKVersion: androidSDKVersion,
				OSName:            osName,
				OSVersion:         osVersion,
			},
		},
		VideoID: videoID,
	}
}

// InnerTube Response DTOs
type InnerTubeResponse struct {
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`

	VideoDetails struct {
		Title         string `json:"title"`
		Author        string `json:"author"`
		LengthSeconds string `json:"lengthSeconds"`
	} `json:"videoDetails"`

	Captions struct {
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []CaptionTrack `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
}

type CaptionTrack struct {
	BaseURL        string `json:"baseUrl"`
	LanguageCode   string `json:"languageCode"`
	Kind           string `json:"kind"`
	IsTranslatable bool   `json:"isTranslatable"`
}

// SelectCaptionTrack finds the best caption track for requested language preference in a single pass.
func SelectCaptionTrack(tracks []CaptionTrack, targetLang string) (*CaptionTrack, bool) {
	if len(tracks) == 0 {
		return nil, false
	}

	targetLang = strings.ToLower(strings.TrimSpace(targetLang))
	if targetLang == "" {
		targetLang = "en"
	}

	var targetManual *CaptionTrack
	var targetASR *CaptionTrack
	var enManual *CaptionTrack
	var enASR *CaptionTrack

	for i := range tracks {
		code := strings.ToLower(tracks[i].LanguageCode)
		isManual := tracks[i].Kind != "asr"

		if strings.HasPrefix(code, targetLang) {
			if isManual && targetManual == nil {
				targetManual = &tracks[i]
			} else if !isManual && targetASR == nil {
				targetASR = &tracks[i]
			}
		} else if targetLang != "en" && strings.HasPrefix(code, "en") {
			if isManual && enManual == nil {
				enManual = &tracks[i]
			} else if !isManual && enASR == nil {
				enASR = &tracks[i]
			}
		}
	}

	switch {
	case targetManual != nil:
		return targetManual, true
	case targetASR != nil:
		return targetASR, true
	case enManual != nil:
		return enManual, true
	case enASR != nil:
		return enASR, true
	default:
		return &tracks[0], true
	}
}

// TimedText JSON3 DTOs
type transcriptJSONResponse struct {
	Events []transcriptEvent `json:"events"`
}

type transcriptEvent struct {
	TStartMs    int64           `json:"tStartMs"`
	DDurationMs int64           `json:"dDurationMs"`
	Segs        []transcriptSeg `json:"segs"`
}

type transcriptSeg struct {
	Utf8 string `json:"utf8"`
}

// BuildTranscriptText parses fmt=json3 payload bytes into formatted transcript paragraphs.
func BuildTranscriptText(data []byte) (string, error) {
	var resp transcriptJSONResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, event := range resp.Events {
		for _, seg := range event.Segs {
			if seg.Utf8 == "\n" {
				sb.WriteString("\n")
				continue
			}
			sb.WriteString(seg.Utf8)
		}
	}

	rawText := sb.String()
	lines := strings.Split(rawText, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	return strings.Join(cleanedLines, "\n"), nil
}

// PrepareJSON3URL converts or appends &fmt=json3 to the caption track baseUrl.
func PrepareJSON3URL(rawURL string) string {
	if strings.Contains(rawURL, "fmt=srv3") {
		return strings.Replace(rawURL, "fmt=srv3", "fmt=json3", 1)
	}
	if !strings.Contains(rawURL, "fmt=json3") {
		if strings.Contains(rawURL, "?") {
			return rawURL + "&fmt=json3"
		}
		return rawURL + "?fmt=json3"
	}
	return rawURL
}
