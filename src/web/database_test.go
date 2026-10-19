package web

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// T011: apertura con reintentos (FR-003, FR-004, SC-002).

type fakePinger struct {
	failures int // cantidad de intentos que fallan antes de responder
	calls    int
	err      error
}

func (p *fakePinger) PingContext(context.Context) error {
	p.calls++
	if p.calls <= p.failures {
		return p.err
	}
	return nil
}

func testPolicy(sleeps *int) RetryPolicy {
	return RetryPolicy{
		Attempts: ConnectAttempts,
		Delay:    ConnectRetryDelay,
		Sleep: func(context.Context, time.Duration) error {
			*sleeps++
			return nil
		},
	}
}

func TestReintentos_ValoresDelPlan(t *testing.T) {
	if ConnectAttempts != 10 || ConnectRetryDelay != 2*time.Second {
		t.Errorf("se esperaban 10 intentos cada 2 s, hay %d cada %v", ConnectAttempts, ConnectRetryDelay)
	}
}

func TestWaitForDatabase_ConectaAlCuartoIntento(t *testing.T) {
	pinger := &fakePinger{failures: 3, err: errors.New("connection refused")}
	sleeps := 0

	err := WaitForDatabase(context.Background(), pinger, testPolicy(&sleeps), slog.New(slog.DiscardHandler), "localhost:5433/db", Config{}.Redact)

	if err != nil {
		t.Fatalf("debía conectar al cuarto intento, se obtuvo: %v", err)
	}
	if pinger.calls != 4 || sleeps != 3 {
		t.Errorf("se esperaban 4 intentos y 3 esperas, hubo %d y %d", pinger.calls, sleeps)
	}
}

func TestWaitForDatabase_AgotaIntentosSinExponerLaClave(t *testing.T) {
	url := "postgres://sym_user:Sup3rS3cret@localhost:5433/db?sslmode=disable"
	cfg := Config{DatabaseURL: url}
	pinger := &fakePinger{failures: 100, err: errors.New("dial " + url + ": password Sup3rS3cret rejected")}
	sleeps := 0
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	err := WaitForDatabase(context.Background(), pinger, testPolicy(&sleeps), logger, DescribeDatabase(url), cfg.Redact)

	var unreachable *DatabaseUnreachableError
	if !errors.As(err, &unreachable) {
		t.Fatalf("se esperaba DatabaseUnreachableError, se obtuvo: %v", err)
	}
	if pinger.calls != 10 || sleeps != 9 {
		t.Errorf("se esperaban 10 intentos y 9 esperas, hubo %d y %d", pinger.calls, sleeps)
	}
	for _, texto := range []string{err.Error(), logs.String()} {
		if strings.Contains(texto, "Sup3rS3cret") || strings.Contains(texto, url) {
			t.Errorf("no debe exponerse la clave ni la URL completa: %q", texto)
		}
	}
	if !strings.Contains(err.Error(), "localhost:5433/db") {
		t.Errorf("el mensaje debe indicar a qué base intentó conectar: %q", err.Error())
	}
}

func TestWaitForDatabase_SeCortaSiSeCancelaElArranque(t *testing.T) {
	pinger := &fakePinger{failures: 100, err: errors.New("connection refused")}
	ctx, cancel := context.WithCancel(context.Background())
	policy := RetryPolicy{Attempts: ConnectAttempts, Delay: time.Hour, Sleep: func(ctx context.Context, d time.Duration) error {
		cancel()
		return ctx.Err()
	}}

	err := WaitForDatabase(ctx, pinger, policy, slog.New(slog.DiscardHandler), "h/db", Config{}.Redact)

	if err == nil || pinger.calls != 1 {
		t.Errorf("debía cortar tras la cancelación; error=%v, intentos=%d", err, pinger.calls)
	}
}
