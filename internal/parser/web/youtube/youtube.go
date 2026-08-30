package youtube

import (
	"context"
	"fmt"
	"net/http"
	"tldr/internal/models"

	"resty.dev/v3"
)

type YtParser struct {
	client *resty.Client
}

func NewYtParser() *YtParser {
	return &YtParser{
		client: newClient(),
	}
}

// YouTubeResult holds the extracted transcript text and video metadata.
type YouTubeResult struct {
	Title      string
	Transcript string
	Language   string
}

// FetchTranscript extracts the video ID, calls InnerTube API, selects the best track,
// fetches the JSON3 transcript, and returns the combined text and title.
func (p *YtParser) FetchTranscript(ctx context.Context, source string, targetLang string) (*YouTubeResult, error) {
	videoID, err := ExtractVideoID(source)
	if err != nil {
		return nil, err
	}

	payload := newInnerTubePayload(videoID)

	var innerTubeResp InnerTubeResponse
	resp, err := p.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", userAgent).
		SetBody(payload).
		SetResult(&innerTubeResp).
		Post(innerTubeAPIURL)

	if err != nil {
		return nil, fmt.Errorf("%w: failed to call InnerTube API: %w", models.ErrUpstreamFailed, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: InnerTube API returned status %d", models.ErrUpstreamFailed, resp.StatusCode())
	}

	if innerTubeResp.PlayabilityStatus.Status != "OK" && innerTubeResp.PlayabilityStatus.Status != "" {
		return nil, fmt.Errorf("%w: status=%s, reason=%s", models.ErrRestricted,
			innerTubeResp.PlayabilityStatus.Status, innerTubeResp.PlayabilityStatus.Reason)
	}

	tracks := innerTubeResp.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
	if len(tracks) == 0 {
		return nil, fmt.Errorf("%w: no captions available for this youtube video", models.ErrNoContent)
	}

	selectedTrack, ok := SelectCaptionTrack(tracks, targetLang)
	if !ok {
		return nil, fmt.Errorf("%w: no suitable caption track found for language %s", models.ErrNoContent, targetLang)
	}

	json3URL := PrepareJSON3URL(selectedTrack.BaseURL)

	transcriptResp, err := p.client.R().
		SetContext(ctx).
		SetHeader("User-Agent", userAgent).
		Get(json3URL)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", models.ErrUpstreamFailed, err)
	}

	if transcriptResp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", models.ErrUpstreamFailed, transcriptResp.StatusCode())
	}

	transcriptText, err := BuildTranscriptText(transcriptResp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse transcript json3: %w", models.ErrNoContent, err)
	}

	if transcriptText == "" {
		return nil, fmt.Errorf("%w: extracted transcript text is empty", models.ErrNoContent)
	}

	return &YouTubeResult{
		Title:      innerTubeResp.VideoDetails.Title,
		Transcript: transcriptText,
		Language:   selectedTrack.LanguageCode,
	}, nil
}
