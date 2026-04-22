package openai

import (
	"fmt"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func BuildBatchGeneratorPrompt(points []domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string) string {
	typeInstruction := ""
	switch questionType {
	case domain.QuestionTypeMultipleChoice:
		typeInstruction = "すべて4択選択問題にしてください。各 questions[i].options には必ず4つの選択肢を含めてください。"
	case domain.QuestionTypeDescriptive:
		typeInstruction = "すべて記述式問題にしてください。各 questions[i].options は空配列にしてください。"
	default:
		typeInstruction = "すべて4択選択問題にしてください。"
	}

	customPart := ""
	if customInstruction != "" {
		customPart = fmt.Sprintf("\n追加指示: %s", customInstruction)
	}

	var pointsSection string
	for index, point := range points {
		pointsSection += fmt.Sprintf("\n%d.\nハイライト本文: %s\nユーザー解説: %s\n", index+1, point.Point, point.Context)
	}

	return fmt.Sprintf(`以下の複数ハイライトから学習用の問題をまとめて作成してください。
各ハイライトにつき1問ずつ作り、順番を保ってください。
「ハイライト本文」を主情報として使い、「ユーザー解説」がある場合は補助情報として使ってください。

素材一覧:%s

%s%s

JSON Schemaに厳密に従うJSONだけを返してください。`, pointsSection, typeInstruction, customPart)
}

func BuildGraderPrompt(question *domain.Question, userAnswer string) string {
	return fmt.Sprintf(`以下の問題に対するユーザーの回答を採点してください。

問題: %s
模範解答: %s
ユーザーの回答: %s

JSON Schemaに厳密に従うJSONだけを返してください。`, question.Content, question.CorrectAnswer, userAnswer)
}
