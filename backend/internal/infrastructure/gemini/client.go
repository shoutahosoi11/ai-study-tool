package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	ModelFlash = "gemini-1.5-flash"
	ModelPro   = "gemini-1.5-pro"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: api key is required")
	}
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * 1000000000,
		},
	}, nil
}

func (c *Client) Close() {
}

func ModelForPlan(plan string) string {
	if plan == "pro" {
		return ModelPro
	}
	return ModelFlash
}

func (c *Client) ExtractPoints(ctx context.Context, text string, model string) ([]domain.ExtractedPoint, error) {
	prompt := BuildExtractorPrompt(text)
	resp, err := c.generate(ctx, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("gemini: extract points failed: %w", err)
	}

	var result struct {
		Points []struct {
			Point   string `json:"point"`
			Context string `json:"context"`
		} `json:"points"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse extract points response: %w", err)
	}

	points := make([]domain.ExtractedPoint, 0, len(result.Points))
	for _, p := range result.Points {
		points = append(points, domain.ExtractedPoint{
			Point:   p.Point,
			Context: p.Context,
		})
	}
	return points, nil
}

func (c *Client) GenerateQuestion(ctx context.Context, point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) (*domain.GeneratedQuestion, error) {
	prompt := BuildGeneratorPrompt(point, questionType, customInstruction)
	resp, err := c.generate(ctx, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("gemini: generate question failed: %w", err)
	}

	var result struct {
		Content       string   `json:"content"`
		Options       []string `json:"options"`
		CorrectAnswer string   `json:"correct_answer"`
		Explanation   string   `json:"explanation"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse generate question response: %w", err)
	}

	return &domain.GeneratedQuestion{
		Content:       result.Content,
		Options:       result.Options,
		CorrectAnswer: result.CorrectAnswer,
		Explanation:   result.Explanation,
	}, nil
}

func (c *Client) GradeAnswer(ctx context.Context, question *domain.Question, userAnswer string, model string) (*domain.GradeResult, error) {
	prompt := BuildGraderPrompt(question, userAnswer)
	resp, err := c.generate(ctx, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("gemini: grade answer failed: %w", err)
	}

	var result struct {
		IsCorrect bool   `json:"is_correct"`
		Score     int    `json:"score"`
		Feedback  string `json:"feedback"`
	}
	if err := parseJSON(resp, &result); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse grade answer response: %w", err)
	}

	return &domain.GradeResult{
		IsCorrect: result.IsCorrect,
		Score:     result.Score,
		Feedback:  result.Feedback,
	}, nil
}

func (c *Client) generate(ctx context.Context, model string, prompt string) (string, error) {
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(model),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gemini: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
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
		if os.Getenv("DEBUG_GEMINI_RAW") != "" {
			return "", fmt.Errorf("gemini: decode response: %w: %s", err, string(respBody))
		}
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}

	var sb strings.Builder
	for _, part := range result.Candidates[0].Content.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String(), nil
}

func parseJSON(raw string, v interface{}) error {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	return json.Unmarshal([]byte(raw), v)
}
