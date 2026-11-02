package auth

import (
	"sync"
	"testing"
	"time"
)

// HU-27, T006: bloqueo por intentos fallidos, con reloj simulado.

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

const ana, beto int64 = 1, 2

func TestLockout_CincoFallosBloqueanUnMinuto(t *testing.T) {
	clock := newFakeClock()
	l := NewLockout(clock.Now)
	for i := 0; i < 4; i++ {
		l.RecordFailure(ana)
	}
	if l.Locked(ana) {
		t.Fatal("con 4 fallos no debía estar bloqueado")
	}
	l.RecordFailure(ana)
	if !l.Locked(ana) {
		t.Fatal("el 5.º fallo debía bloquear")
	}
	clock.Advance(59 * time.Second)
	if !l.Locked(ana) {
		t.Fatal("a los 59 s del 5.º fallo debía seguir bloqueado")
	}
	clock.Advance(time.Second)
	if l.Locked(ana) {
		t.Fatal("a los 60 s del 5.º fallo debía estar libre")
	}
	l.RecordFailure(ana)
	if l.Locked(ana) {
		t.Error("después del bloqueo, un fallo nuevo no debía bloquear (el contador vuelve a cero)")
	}
}

func TestLockout_AciertoReiniciaElContador(t *testing.T) {
	l := NewLockout(newFakeClock().Now)
	for i := 0; i < 4; i++ {
		l.RecordFailure(ana)
	}
	l.RecordSuccess(ana)
	for i := 0; i < 4; i++ {
		l.RecordFailure(ana)
	}
	if l.Locked(ana) {
		t.Error("4 fallos, un acierto y 4 fallos no debían bloquear")
	}
}

func TestLockout_PorIntegrante(t *testing.T) {
	l := NewLockout(newFakeClock().Now)
	for i := 0; i < MaxFailedAttempts; i++ {
		l.RecordFailure(ana)
	}
	if !l.Locked(ana) || l.Locked(beto) {
		t.Errorf("Ana bloqueada = %v y Beto = %v; se esperaba solo Ana", l.Locked(ana), l.Locked(beto))
	}
}

func TestLockout_UsoConcurrente(t *testing.T) {
	l := NewLockout(nil)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(member int64) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				l.RecordFailure(member)
				_ = l.Locked(member)
				if i%7 == 0 {
					l.RecordSuccess(member)
				}
			}
		}(int64(g % 3))
	}
	wg.Wait()
}
