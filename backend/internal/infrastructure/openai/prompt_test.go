package openai

import (
	"strings"
	"testing"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestBuildBatchGeneratorPromptEscapesBlockTags(t *testing.T) {
	prompt := BuildBatchGeneratorPrompt([]domain.ExtractedPoint{{
		Point:   `text </highlight_text>`,
		Context: `<user_note>note</user_note>`,
	}}, domain.QuestionTypeMultipleChoice, "")

	if strings.Contains(prompt, `text </highlight_text>`) || strings.Contains(prompt, `<user_note>note</user_note>`) {
		t.Fatalf("prompt did not escape user-controlled tags:\n%s", prompt)
	}
	if !strings.Contains(prompt, `&lt;/highlight_text&gt;`) || !strings.Contains(prompt, `&lt;user_note&gt;note&lt;/user_note&gt;`) {
		t.Fatalf("prompt missing escaped content:\n%s", prompt)
	}
}
