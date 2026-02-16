package store

import (
	"context"
	"database/sql"
)

type PostStore interface {
	Create(ctx context.Context, post *Post) error
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