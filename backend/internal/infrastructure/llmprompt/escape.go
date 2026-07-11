package llmprompt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxCustomInstructionLength = 500

func EscapeBlockText(value string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(value)
}

// BuildCustomInstructionBlock はユーザー由来の追加指示をエスケープして
// タグで囲む。プロンプト本文への直接連結を許すとインジェクション経路に
// なるため、呼び出し側はこの関数を必ず経由する。
func BuildCustomInstructionBlock(customInstruction string) (string, error) {
	trimmed := strings.TrimSpace(customInstruction)
	if trimmed == "" {
		return "", nil
	}
	if utf8.RuneCountInString(trimmed) > MaxCustomInstructionLength {
		return "", fmt.Errorf("custom instruction must be at most %d characters", MaxCustomInstructionLength)
	}

	return fmt.Sprintf("\n<custom_instruction>\n%s\n</custom_instruction>", EscapeBlockText(trimmed)), nil
}
