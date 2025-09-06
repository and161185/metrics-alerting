package utils

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// WithRetry runs the given function with retry logic.
// Retries up to 3 times with delays: 1s, 3s, and 5s.
func WithRetry(ctx context.Context, fn func() error) error {
	delays := []int{1, 3, 5}
	var err error
	for _, delay := range delays {
		err = fn()
		if err == nil || !isRetriable(err) {
			return err
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}
	return err
}

func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	// не ретраим отмену/дедлайн
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// PG-коды
	retriableCodes := map[string]struct{}{
		pgerrcode.ConnectionException:                           {},
		pgerrcode.ConnectionDoesNotExist:                        {},
		pgerrcode.ConnectionFailure:                             {},
		pgerrcode.SQLClientUnableToEstablishSQLConnection:       {},
		pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection: {},
		pgerrcode.TransactionResolutionUnknown:                  {},
		pgerrcode.SerializationFailure:                          {},
		pgerrcode.TooManyConnections:                            {},
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		_, ok := retriableCodes[pgErr.Code]
		return ok
	}

	// сетевые — ретраим
	var nerr net.Error
	if errors.As(err, &nerr) {
		return true
	}

	// прочие таймауты (например os.ErrDeadlineExceeded)
	if os.IsTimeout(err) {
		return true
	}
	return false
}
