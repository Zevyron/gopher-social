package store

import (
	"context"
	"database/sql"
)

// InsertOutboxEvent records an event inside an ongoing transaction.
// It accepts *sql.Tx directly so any repository method can participate in the outbox.
func InsertOutboxEvent(ctx context.Context, tx *sql.Tx, eventType string, payload []byte) error {
	query := `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1, $2)
	`

	_, err := tx.ExecContext(ctx, query, eventType, payload)
	return err
}