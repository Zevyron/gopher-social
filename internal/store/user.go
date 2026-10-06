package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID 				int64		`json:"id"`
	Username 		string 		`json:"username"`
	FirstName		string		`json:"first_name"`
	LastName		string		`json:"last_name"`
	Email			string		`json:"email"`
	Password		password	`json:"-"`
	CreatedAt		string		`json:"created_at"`
	UpdatedAt		string		`json:"updated_at"`
	IsActive		bool		`json:"is_active"`
}

type password struct {
	text 	*string
	hash	[]byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.text = &text
	p.hash = hash

	return nil
}

type PostgresUserStore struct {
	db *sql.DB
}

var (
	ErrDuplicateEmail    = errors.New("a user with that email already exists")
	ErrDuplicateUsername = errors.New("a user with that username already exists")
	ErrInvalidOrExpiredToken = errors.New("invalid or expired token")
	ErrUserNotFoundOrActive = errors.New("user not found or already active")
)

type OutboxEventPayload struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (u *PostgresUserStore) Create(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users(username, first_name, last_name, email, password) VALUES($1, $2, $3, $4, $5) RETURNING id, created_at, is_active
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, user.Username, user.FirstName, user.LastName, user.Email, user.Password.hash)
	} else {
		row = u.db.QueryRowContext(ctx, query, user.Username, user.FirstName, user.LastName, user.Email, user.Password.hash)
	}

	err := row.Scan(&user.ID, &user.CreatedAt, &user.IsActive)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}

	return nil
}

func (u *PostgresUserStore) GetByID(ctx context.Context, userID int64) (*User, error) {
	query := `
		SELECT id, username, first_name, last_name, email, password, created_at
		FROM users
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &User{}
	err := u.db.QueryRowContext(ctx, query, userID).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.CreatedAt)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return user, nil

}

func (u *PostgresUserStore) CreateAndInvite(ctx context.Context, user *User, plainToken string, tokenHash []byte, invitationExp time.Duration) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		if err := u.Create(ctx, tx, user); err != nil {
			return err
		}

		if err := u.createUserInvitation(ctx, tx, tokenHash, invitationExp, user.ID); err != nil {
			return err // Return err so withTx triggers a rollback
		};

		payload, err := json.Marshal(OutboxEventPayload{
			UserID:   user.ID,
			Email:    user.Email,
			Username: user.Username,
			Token:    plainToken,
		})
		if err != nil {
			return err
		}

		if err := u.createOutboxEvent(ctx, tx, "user.created", payload); err != nil {
			return err
		}

		return nil
	})
}

func (u *PostgresUserStore) createOutboxEvent(ctx context.Context, tx *sql.Tx, eventType string, payload []byte) error {
	query := `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1, $2)
	`

	_, err := tx.ExecContext(ctx, query, eventType, payload)
	return err
}

func (u *PostgresUserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, tokenHash []byte, invitationExp time.Duration, userID int64) error {
	query := `INSERT INTO user_invitations (token, user_id, expiry) VALUES ($1, $2, $3)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, tokenHash, userID, time.Now().Add(invitationExp))
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresUserStore) Activate(ctx context.Context, plainToken string) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		// Hash the plaintext token to match the BYTEA column
		hashArr := sha256.Sum256([]byte(plainToken))
		tokenHash := hashArr[:]

		// Identify user and ensure token has not expired
		queryFind := `
			SELECT user_id 
			FROM user_invitations 
			WHERE token = $1 AND expiry > NOW()
		`
		var userID int64
		err := tx.QueryRowContext(ctx, queryFind, tokenHash).Scan(&userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidOrExpiredToken
			}
			return err
		}

		// Activate user
		queryActivate := `
			UPDATE users 
			SET is_active = true 
			WHERE id = $1
		`
		if _, err := tx.ExecContext(ctx, queryActivate, userID); err != nil {
			return err
		}

		// Invalidate token so it cannot be reused
		queryDeleteToken := `
			DELETE FROM user_invitations 
			WHERE user_id = $1
		`
		_, err = tx.ExecContext(ctx, queryDeleteToken, userID)
		return err
	})
}

func (s *PostgresUserStore) ResendInvitation(ctx context.Context, email, plainToken string, tokenHash []byte, invitationExp time.Duration) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		queryUser := `
			SELECT id, username, is_active
			FROM users
			WHERE email = $1
		`
		var user User
		err := tx.QueryRowContext(ctx, queryUser, email).Scan(&user.ID, &user.Username, &user.IsActive)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFoundOrActive
			}
			return err
		}
		if user.IsActive {
			return ErrUserNotFoundOrActive
		}

		queryDeleteOldTokens := `
			DELETE FROM user_invitations
			WHERE user_id = $1
		`
		if _, err := tx.ExecContext(ctx, queryDeleteOldTokens, user.ID); err != nil {
			return err
		}

		queryInsertToken := `
			INSERT INTO user_invitations (token, user_id, expiry)
			VALUES ($1, $2, $3)
		`
		if _, err := tx.ExecContext(ctx, queryInsertToken, tokenHash, user.ID, time.Now().Add(invitationExp)); err != nil {
			return err
		}

		payload, err := json.Marshal(OutboxEventPayload{
			UserID: user.ID,
			Email: email,
			Username: user.Username,
			Token: plainToken,
		})
		if err != nil {
			return err
		}

		return InsertOutboxEvent(ctx, tx, "user.created", payload)
		})
}