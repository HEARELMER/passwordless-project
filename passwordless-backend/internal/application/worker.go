package application

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func StartSessionCleanupWorker(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("session cleanup worker stopped")
				return
			case <-ticker.C:
				cleanupExpiredSessions(ctx, pool)
			}
		}
	}()
}

func cleanupExpiredSessions(ctx context.Context, pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const query = `
		UPDATE webauthn_sessions 
		SET status = 'EXPIRED' 
		WHERE expires_at < NOW() AND status = 'PENDING'
	`
	tag, err := pool.Exec(ctx, query)
	if err != nil {
		log.Printf("[Worker] error cleaning expired sessions: %v", err)
		return
	}

	if tag.RowsAffected() > 0 {
		log.Printf("[Worker] expired sessions marked: %d", tag.RowsAffected())
	}
}
