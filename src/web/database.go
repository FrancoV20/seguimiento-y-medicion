package web

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"
)

// Reintentos de conexión al arrancar (HU-20a, T018).
const (
	ConnectAttempts   = 10
	ConnectRetryDelay = 2 * time.Second
)

// Pinger es lo mínimo que se necesita para comprobar la conexión (lo cumple *sql.DB).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// RetryPolicy define cuántas veces y cada cuánto se reintenta.
type RetryPolicy struct {
	Attempts int
	Delay    time.Duration
	Sleep    func(ctx context.Context, d time.Duration) error
}

// DefaultRetryPolicy son 10 intentos con 2 s de espera.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Attempts: ConnectAttempts, Delay: ConnectRetryDelay, Sleep: sleepContext}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// DatabaseUnreachableError indica que se agotaron los reintentos.
// Target es la descripción sin credenciales (ver DescribeDatabase).
type DatabaseUnreachableError struct {
	Target   string
	Attempts int
}

func (e *DatabaseUnreachableError) Error() string {
	return fmt.Sprintf("no se pudo conectar a la base de datos en %s después de %d intentos: verificá que PostgreSQL esté levantado (docker compose up -d) y que DATABASE_URL tenga el usuario, la clave y el puerto correctos",
		e.Target, e.Attempts)
}

// WaitForDatabase reintenta el ping hasta que responde o se agota la política.
// redact limpia los errores del driver antes de registrarlos.
func WaitForDatabase(ctx context.Context, p Pinger, policy RetryPolicy, logger *slog.Logger, target string, redact func(string) string) error {
	if policy.Sleep == nil {
		policy.Sleep = sleepContext
	}
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		err := p.PingContext(ctx)
		if err == nil {
			if attempt > 1 {
				logger.Info("conexión a la base de datos establecida", "destino", target, "intento", attempt)
			}
			return nil
		}
		logger.Warn("la base de datos todavía no responde",
			"destino", target, "intento", attempt, "de", policy.Attempts, "detalle", redact(err.Error()))
		if attempt == policy.Attempts {
			break
		}
		if err := policy.Sleep(ctx, policy.Delay); err != nil {
			return fmt.Errorf("arranque cancelado mientras se esperaba la base de datos")
		}
	}
	return &DatabaseUnreachableError{Target: target, Attempts: policy.Attempts}
}

// OpenDatabase abre el pool de conexiones y espera a que la base responda.
func OpenDatabase(ctx context.Context, cfg Config, policy RetryPolicy, logger *slog.Logger) (*sql.DB, error) {
	target := DescribeDatabase(cfg.DatabaseURL)
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL no tiene un formato válido (destino: %s)", target)
	}
	if err := WaitForDatabase(ctx, db, policy, logger, target, cfg.Redact); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
