package store

import (
	"context"
	"database/sql"
)

type Follower struct {
	UserID     int64  `json:"user_id"`
	FollowerID int64  `json:"follow_id"`
	CreatedAt  string `json:"created_at"`
}

type PostgresFollowerStore struct {
	db *sql.DB
}

func (s *PostgresFollowerStore) Follow(ctx context.Context, userID, followerID int64) error {
	query := `
		INSERT INTO followers (user_id, follower_id) VALUES($1, $2)
		ON CONFLICT (user_id, follower_id) DO NOTHING
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, followerID)
	return err
}

func (s *PostgresFollowerStore) Unfollow(ctx context.Context, userID, followerID int64) error {
	query := `
		DELETE FROM followers
		WHERE user_id = $1 AND follower_id = $2
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, followerID)
	return err
} 