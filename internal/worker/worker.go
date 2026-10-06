package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Zevyron/gopher-social/internal/mailer"
	"github.com/Zevyron/gopher-social/internal/store"
	"go.uber.org/zap"
)

type OutboxEvent struct {
	ID         string
	EventType  string
	Payload    []byte
	RetryCount int
}

type OutboxWorker struct {
	db           *sql.DB
	mailer       mailer.Client
	logger       *zap.SugaredLogger
	frontendURL  string
	pollInterval time.Duration
	batchSize    int
	maxRetries   int
}

func NewOutboxWorker(
	db *sql.DB,
	m mailer.Client,
	logger *zap.SugaredLogger,
	frontendURL string,
) *OutboxWorker {
	return &OutboxWorker{
		db:           db,
		mailer:       m,
		logger:       logger,
		frontendURL:  frontendURL,
		pollInterval: 1 * time.Second,
		batchSize:    10,
		maxRetries:   5,
	}
}

// Start runs the polling loop until ctx is canceled.
func (w *OutboxWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.logger.Infow("outbox worker started", "poll_interval", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker shutting down gracefully...")
			return
		case <-ticker.C:
			events, err := w.claimBatch(ctx)
			if err != nil {
				w.logger.Errorw("failed to claim outbox batch", "error", err)
				continue
			}

			for _, event := range events {
				w.processEvent(ctx, event)
			}
		}
	}
}

// claimBatch atomically claims up to batchSize events using a short-lived transaction.
func (w *OutboxWorker) claimBatch(ctx context.Context) ([]OutboxEvent, error) {
	// 1-minute lease: If this worker crashes mid-network call, another worker can reclaim it later.
	query := `
		WITH ready_events AS (
			SELECT id
			FROM outbox_events
			WHERE (status = 'pending' OR (status = 'processing' AND next_run_at <= NOW()))
				AND next_run_at <= NOW()
			ORDER BY next_run_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events o
		SET status = 'processing',
			next_run_at = NOW() + INTERVAL '1 minute'
		FROM ready_events r
		WHERE o.id = r.id
		RETURNING o.id, o.event_type, o.payload, o.retry_count;
	`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := w.db.QueryContext(ctx, query, w.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.RetryCount); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

func (w *OutboxWorker) processEvent(ctx context.Context, event OutboxEvent) {
	var err error

	switch event.EventType {
	case "user.created":
		err = w.handleUserCreated(ctx, event.Payload)
	default:
		err = fmt.Errorf("unhandled event_type: %s", event.EventType)
	}

	if err != nil {
		w.logger.Errorw("failed processing outbox event", "event_id", event.ID, "error", err)
		w.scheduleRetry(ctx, event, err)
		return
	}

	w.markCompleted(ctx, event.ID)
}

func (w *OutboxWorker) handleUserCreated(ctx context.Context, rawPayload []byte) error {
	var p store.OutboxEventPayload
	if err := json.Unmarshal(rawPayload, &p); err != nil {
		return fmt.Errorf("malformed user.created payload: %w", err)
	}

	activationURL := fmt.Sprintf("%s/v1/authentication/activate?token=%s", w.frontendURL, p.Token)
	data := map[string]any{
		"Username":      p.Username,
		"ActivationURL": activationURL,
	}

	return w.mailer.Send(p.Email, "user_invitation.tmpl", data)
}

func (w *OutboxWorker) markCompleted(ctx context.Context, eventID string) {
	query := `
		UPDATE outbox_events
		SET status = 'completed',
			processed_at = NOW()
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := w.db.ExecContext(ctx, query, eventID); err != nil {
		w.logger.Errorw("failed marking event completed", "event_id", eventID, "error", err)
	}
}

func (w *OutboxWorker) scheduleRetry(ctx context.Context, event OutboxEvent, dispatchErr error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Dead-letter queue transition on retry exhaustion
	if event.RetryCount >= w.maxRetries {
		w.logger.Warnw("outbox event exhausted retries; dead-lettering", "event_id", event.ID)
		query := `UPDATE outbox_events SET status = 'dead_letter' WHERE id = $1`
		_, _ = w.db.ExecContext(ctx, query, event.ID)
		return
	}

	// Exponential backoff: 2s, 4s, 8s, 16s, 32s
	backoffSeconds := math.Pow(2, float64(event.RetryCount+1))
	nextRun := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	query := `
		UPDATE outbox_events
		SET status = 'pending',
			retry_count = retry_count + 1,
			next_run_at = $1
		WHERE id = $2
	`
	_, _ = w.db.ExecContext(ctx, query, nextRun, event.ID)
}