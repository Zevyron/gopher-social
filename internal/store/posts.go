package store

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type Post struct {
	ID 				int64		`json:"id"`
	Content 		string		`json:"context"`
	Title 			string		`json:"title"`
	UserID 			int64		`json:"user_id"`
	Tags			[]string	`json:"tags"`
	CreatedAt		string		`json:"created_at"`
	UpdatedAt		string		`json:"updated_at"`
	DeletedAt		string		`json:"deleted_at"`
}

type PostgresPostsStore struct { // can omit postgres since only postgres is being used
	db *sql.DB
}

func (s *PostgresPostsStore) Create(ctx context.Context, post *Post) error {
	query := `
		INSERT INTO posts (content, title, user_id, tags)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at	
	`

	err := s.db.QueryRowContext(ctx, query, post.Content, post.Title, post.UserID, post.Tags, pq.Array(post.Tags)).Scan(
		&post.ID, &post.CreatedAt, &post.UpdatedAt, &post.DeletedAt,
	)

	if err != nil {
		return err
	}

	return nil
}