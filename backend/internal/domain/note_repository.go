package domain

import "context"

type NoteRepository interface {
	Save(ctx context.Context, note *Note) error
}
