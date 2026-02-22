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
}

type UserStore interface {
	Create(ctx context.Context, user *User) error
}

type Storage struct {
	Posts PostStore
	Users UserStore

}

func NewPostgresStorage(db *sql.DB) *Storage {
	return &Storage{
		Posts: &PostgresPostsStore{db},
		Users: &PostgresUsersStore{db},
	}
}