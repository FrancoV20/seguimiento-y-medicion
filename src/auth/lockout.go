package auth

import (
	"sync"
	"time"
)

const (
	// MaxFailedAttempts fallos seguidos bloquean al integrante durante LockDuration.
	MaxFailedAttempts = 5
	LockDuration      = time.Minute
)

// Lockout cuenta los intentos fallidos seguidos de cada integrante y lo bloquea un minuto al
// llegar a MaxFailedAttempts. Vive en memoria: se reinicia con el servidor (limitación
// documentada). Es seguro para uso concurrente.
type Lockout struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[int64]*lockoutEntry
}

type lockoutEntry struct {
	failures    int
	lockedUntil time.Time // cero si no está bloqueado
}

// NewLockout crea el bloqueo con el reloj indicado; nil usa time.Now.
func NewLockout(now func() time.Time) *Lockout {
	if now == nil {
		now = time.Now
	}
	return &Lockout{now: now, entries: map[int64]*lockoutEntry{}}
}

// Locked indica si el integrante está bloqueado. Si el bloqueo venció, lo libera y reinicia el
// contador.
func (l *Lockout) Locked(memberID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[memberID]
	if !ok || e.lockedUntil.IsZero() {
		return false
	}
	if !l.now().Before(e.lockedUntil) {
		delete(l.entries, memberID)
		return false
	}
	return true
}

// RecordFailure suma un intento fallido; al llegar a MaxFailedAttempts bloquea al integrante.
func (l *Lockout) RecordFailure(memberID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[memberID]
	if !ok {
		e = &lockoutEntry{}
		l.entries[memberID] = e
	}
	e.failures++
	if e.failures >= MaxFailedAttempts && e.lockedUntil.IsZero() {
		e.lockedUntil = l.now().Add(LockDuration)
	}
}

// RecordSuccess reinicia el contador del integrante.
func (l *Lockout) RecordSuccess(memberID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, memberID)
}
