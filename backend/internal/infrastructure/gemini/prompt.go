package gemini

import (
	"fmt"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func BuildExtractorPrompt(text string) string {
	return fmt.Sprintf(`以下のテキストから学習に重要なポイントを最大5つ抽出してください。
各ポイントは問題を作れるほど具体的である必要があります。

テキスト:
%s

以下のJSON形式で回答してください。他のテキストは含めないでください:
{
  "points": [
    {
      "point": "重要なポイントの説明",
      "context": "このポイントが登場する文脈や背景"
    }
  ]
}`, text)
}

func BuildGeneratorPrompt(point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string) string {
	typeInstruction := ""
	switch questionType {
	case domain.QuestionTypeMultipleChoice:
		typeInstruction = "4択選択問題を作成してください。optionsには必ず4つの選択肢を含めてください。"
	case domain.QuestionTypeDescriptive:
		typeInstruction = "記述式問題を作成してください。optionsは空配列にしてください。"
	default:
		typeInstruction = "4択選択問題を作成してください。"
	}

	customPart := ""
	if customInstruction != "" {
		customPart = fmt.Sprintf("\n追加指示: %s", customInstruction)
	}

	return fmt.Sprintf(`以下のポイントから学習用の問題を1問作成してください。

ポイント: %s
文脈: %s

%s%s

以下のJSON形式で回答してください。他のテキストは含めないでください:
{
  "content": "問題文",
  "options": ["選択肢A", "選択肢B", "選択肢C", "選択肢D"],
  "correct_answer": "正解の選択肢（選択問題の場合）または模範解答（記述の場合）",
  "explanation": "解説文"
}`, point.Point, point.Context, typeInstruction, customPart)
}

func BuildGraderPrompt(question *domain.Question, userAnswer string) string {
	return fmt.Sprintf(`以下の問題に対するユーザーの回答を採点してください。

問題: %s
模範解答: %s
ユーザーの回答: %s

以下のJSON形式で回答してください。他のテキストは含めないでください:
{
  "is_correct": true/false,
  "score": 0-100の整数,
  "feedback": "フィードバックコメント"
}`, question.Content, question.CorrectAnswer, userAnswer)
}
