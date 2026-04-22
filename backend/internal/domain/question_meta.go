package domain

type QuestionMeta struct {
	QuestionID    string
	CreatorID     string
	SourceType    SourceType
	SourceID      string
	HighlightID   string
	GenerationID  string
	IsAIGenerated bool
}
