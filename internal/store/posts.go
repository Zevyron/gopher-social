package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

type Post struct {
	ID 				int64		`json:"id"`
	Content 		string		`json:"content"`
	Title 			string		`json:"title"`
	UserID 			int64		`json:"user_id"`
	Tags			[]string	`json:"tags"`
	CreatedAt		string		`json:"created_at"`
	UpdatedAt		string		`json:"updated_at"`
	DeletedAt		string		`json:"deleted_at"`
	Version			int64		
	Comments		[]Comment	`json:"comments"`
	User 			User		`json:"user"`
}

type PostWithMetadata struct {
	Post
	CommentCount int `json:"comments_count"`
}

type PostgresPostStore struct { // can omit postgres since only postgres is being used
	db *sql.DB
}

func (s *PostgresPostStore) Create(ctx context.Context, post *Post) error {
	query := `
		INSERT INTO posts (content, title, user_id, tags)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at	
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, post.Content, post.Title, post.UserID, pq.Array(post.Tags)).Scan(
		&post.ID, &post.CreatedAt, &post.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresPostStore) GetByID(ctx context.Context, postID int64) (*Post, error) {
	query := `
		SELECT id, user_id, title, content, created_at, updated_at, tags, version FROM posts WHERE id = $1	
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var post Post
	err := s.db.QueryRowContext(ctx, query, postID).Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt, &post.UpdatedAt, pq.Array(post.Tags), &post.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return &post, nil
}

func (s *PostgresPostStore) DeleteByPostID(ctx context.Context, postID int64) (error) {
	query := `
		DELETE FROM posts WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	result, err := s.db.ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("error executing delete for post id %d: %w", postID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error for checking rows affected for delete: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no post deleted for id %d: %w", postID, errors.New("record not found"))
	}

	return nil
}

func(s *PostgresPostStore) Update(ctx context.Context, post *Post) error {
	query := `
		UPDATE posts
		SET title = $1, content = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, post.Title, post.Content, post.ID, post.Version).Scan(&post.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNotFound
		default:
			return err
		}
	}

	return nil
}

func (s *PostgresPostStore) GetUserFeed(ctx context.Context, userID int64, fq PaginatedFeedQuery) ([]PostWithMetadata, error) {

	sortDir := "DESC"
	cursorDir := "<"

	if fq.Sort == "asc" {
		sortDir = "ASC"
		cursorDir = ">"
	}

	timestamp, cursorID, err := DecodeCursor(fq.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	args := []any{userID, timestamp, cursorID}


	query := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, p.version, p.tags,
			u.username,
			COUNT(c.id) AS comments_count
		FROM posts p
		LEFT JOIN comments c ON c.post_id = p.id
		LEFT JOIN users u ON p.user_id = u.id
		WHERE (p.user_id = $1 OR p.user_id IN (
			SELECT user_id FROM followers WHERE follower_id = $1
		))
		AND (
			p.created_at %s $2 OR 
			(p.created_at = $2 AND p.id %s $3)
		)
	`, cursorDir, cursorDir)

	if len(fq.Tags) > 0 {
		args = append(args, pq.Array(fq.Tags))
		query += fmt.Sprintf(" AND p.tags @> $%d", len(args))
	}

	if fq.Search != "" {
		args = append(args, "%"+fq.Search+"%")
		searchParam := fmt.Sprintf("$%d", len(args))
		query += fmt.Sprintf(" AND (p.title ILIKE %s OR p.content ILIKE %s)", searchParam, searchParam)
	}

	if fq.Since != "" {
		args = append(args, fq.Since)
		query += fmt.Sprintf(" AND p.created_at >= $%d", len(args))
	}

	if fq.Until != "" {
		args = append(args, fq.Until)
		query += fmt.Sprintf(" AND p.created_at <= $%d", len(args))
	}

	args = append(args, fq.Limit)
	query += fmt.Sprintf(`
		GROUP BY p.id, u.username
		ORDER BY p.created_at ` + sortDir + `, p.id DESC
		LIMIT $%d;
	`, len(args))

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var feed []PostWithMetadata
	for rows.Next() {
		var p PostWithMetadata
		err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Title,
			&p.Content,
			&p.CreatedAt,
			&p.Version,
			pq.Array(&p.Tags),
			&p.User.Username,
			&p.CommentCount,
		)
		if err = rows.Err();  err != nil {
			return nil, err
		}

		feed = append(feed, p)
	}
	

	return feed, nil
}