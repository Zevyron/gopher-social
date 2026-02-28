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
	INSERT INTO users(username, first_name, last_name, email, password) VALUES($1, $2, $3) RETURNING id, created_at
	`

	err := u.db.QueryRowContext(ctx, query, user.Username, user.FirstName, user.LastName, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return err
	}

	return nil
}