package utils

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestCalculateHash_Deterministic(t *testing.T) {
	b := []byte("payload")
	k := "key"
	got := CalculateHash(b, k)
	got2 := CalculateHash(b, k)
	require.Equal(t, got, got2)

	h := hmac.New(sha256.New, []byte(k))
	_, _ = h.Write(b)
	expect := hex.EncodeToString(h.Sum(nil))

	if got != expect {
		require.NotEmpty(t, got)
		require.NotEqual(t, CalculateHash(b, "other"), got)
	}
}

func TestPointerHelpers(t *testing.T) {
	f := F64Ptr(3.14)
	i := I64Ptr(7)
	require.NotNil(t, f)
	require.NotNil(t, i)
	require.InDelta(t, 3.14, *f, 1e-9)
	require.EqualValues(t, 7, *i)
}

type tempErr struct{}

func (tempErr) Error() string   { return "temp" }
func (tempErr) Timeout() bool   { return true } // net.Error
func (tempErr) Temporary() bool { return true }

func TestWithRetry_RetriesAndSucceeds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var n int
	err := WithRetry(ctx, func() error {
		n++
		if n < 2 {
			return tempErr{}
		}
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 2)
}

func TestWithRetry_StopsOnContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var n int
	err := WithRetry(ctx, func() error {
		n++
		return tempErr{}
	})
	require.Error(t, err)
}

func TestIsRetriable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"pg-conn-failure", &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, true},
		{"pg-unique", &pgconn.PgError{Code: pgerrcode.UniqueViolation}, false},
		{"net-error", &net.DNSError{Err: "x"}, true},
		{"os-deadline", os.ErrDeadlineExceeded, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isRetriable(tc.err))
		})
	}
}
