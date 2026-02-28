package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("resource not found")
	QueryTimeoutDuration = time.Second*5
)

type PostStore interface {
	Create(ctx context.Context, post *Post) error
	GetByID(ctx context.Context, postID int64) (*Post, error)
	DeleteByPostID(ctx context.Context, postID int64) error
	Update(ctx context.Context, post *Post) error
}

type UserStore interface {
	GetByID(ctx context.Context, userID int64) (*User, error)
	Create(ctx context.Context, user *User) error
}

type CommentStore interface {
	Create(ctx context.Context, comment *Comment) error
	GetByPostID(ctx context.Context, postID int64) ([]Comment, error)
	
}

type FollowStore interface {
	Follow(ctx context.Context, followUser int64, userID int64) error
	Unfollow(ctx context.Context, followUser int64, userID int64) error
}

type Storage struct {
	Posts PostStore
	Users UserStore
	Comments CommentStore
	Followers FollowStore

}

func NewPostgresStorage(db *sql.DB) *Storage {
	return &Storage{
		Posts: &PostgresPostStore{db},
		Users: &PostgresUserStore{db},
		Comments: &PostgresCommentStore{db},
		Followers: &PostgresFollowerStore{db},
	}
}