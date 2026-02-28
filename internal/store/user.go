package store

import (
	"context"
	"database/sql"
)

type User struct {
	ID 				int64		`json:"id"`
	Username 		string 		`json:"username"`
	FirstName		string		`json:"first_name"`
	LastName		string		`json:"last_name"`
	Email			string		`json:"email"`
	Password		string		`json:"-"`
	CreatedAt		string		`json:"created_at"`
	UpdatedAt		string		`json:"updated_at"`
}

type PostgresUserStore struct {
	db *sql.DB
}

func (u *PostgresUserStore) Create(ctx context.Context, user *User) error {
	query := `
	INSERT INTO users(username, first_name, last_name, email, password) VALUES($1, $2, $3, $4, $5) RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, user.Username, user.FirstName, user.LastName, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return err
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