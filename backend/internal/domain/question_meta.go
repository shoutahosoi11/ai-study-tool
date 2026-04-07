package domain

type QuestionMeta struct {
	QuestionID    string
	CreatorID     string
	SourceType    SourceType
	SourceID      string
	GenerationID  string
	IsAIGenerated bool
}
