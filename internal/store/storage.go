package store

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrNotFound = errors.New("resource not found")
)

type PostStore interface {
	Create(ctx context.Context, post *Post) error
	GetByID(ctx context.Context, postID int64) (*Post, error)
	DeleteByPostID(ctx context.Context, postID int64) error
	Update(ctx context.Context, post *Post) error
}

type UserStore interface {
	Create(ctx context.Context, user *User) error
}

type CommentStore interface {
	GetByPostID(ctx context.Context, postID int64) ([]Comment, error)
}

type Storage struct {
	Posts PostStore
	Users UserStore
	Comments CommentStore

}

func NewPostgresStorage(db *sql.DB) *Storage {
	return &Storage{
		Posts: &PostgresPostStore{db},
		Users: &PostgresUserStore{db},
		Comments: &PostgresCommentStore{db},
	}
}