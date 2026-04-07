package gemini

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type OCRClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewOCRClient(apiKey string) (*OCRClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini ocr: api key is required")
	}
	return &OCRClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * 1000000000,
		},
	}, nil
}

func (o *OCRClient) ExtractText(ctx context.Context, imageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: create image request: %w", err)
	}

	imageResp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: fetch image failed: %w", err)
	}
	defer imageResp.Body.Close()

	imageBytes, err := io.ReadAll(imageResp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: read image failed: %w", err)
	}

	mimeType := imageResp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": "この画像のテキストを全て正確に抽出してください。レイアウトを保持し、日本語・英語どちらも対応してください。",
					},
					{
						"inlineData": "",
					},
				},
			},
		},
	}

	payload = map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []interface{}{
					map[string]string{
						"text": "この画像のテキストを全て正確に抽出してください。レイアウトを保持し、日本語・英語どちらも対応してください。",
					},
					map[string]string{
						"mimeType": mimeType,
						"data":     base64.StdEncoding.EncodeToString(imageBytes),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: marshal request: %w", err)
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(ModelFlash),
		url.QueryEscape(o.apiKey),
	)
	ocrReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini ocr: create request: %w", err)
	}
	ocrReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(ocrReq)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: extract text failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini ocr: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gemini ocr: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("gemini ocr: decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini ocr: empty response")
	}

	var sb strings.Builder
	for _, part := range result.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
}
