package web

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func quietLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// T014: apagado ordenado con una solicitud en curso (FR-007).

func TestServe_ApagadoOrdenadoCompletaLaSolicitudEnCurso(t *testing.T) {
	s := NewServer(quietLogger())
	started, release := make(chan struct{}), make(chan struct{})
	s.HandleFunc("GET /lento", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "completada")
	})
	ln, err := s.Listen(freeAddr(t))
	if err != nil {
		t.Fatalf("no se pudo escuchar: %v", err)
	}
	addr := ln.Addr().String()
	ctx, stop := context.WithCancel(context.Background())
	var dbClosed atomic.Bool
	served := make(chan error, 1)
	go func() { served <- s.Serve(ctx, ln, func() { dbClosed.Store(true) }) }()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/lento")
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		got <- result{body: string(b)}
	}()
	<-started

	stop() // orden de detención con la solicitud en curso
	if !esperar(2*time.Second, func() bool {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			c.Close()
		}
		return err != nil
	}) {
		t.Error("después de la orden de detención no debe aceptar conexiones nuevas")
	}
	if dbClosed.Load() {
		t.Error("la base no debe cerrarse mientras hay solicitudes en curso")
	}

	close(release)
	r := <-got
	if r.err != nil || r.body != "completada" {
		t.Errorf("la solicitud en curso debía completarse, se obtuvo %q (error: %v)", r.body, r.err)
	}
	select {
	case err := <-served:
		if err != nil {
			t.Errorf("el apagado debía terminar sin error, se obtuvo: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve no terminó después del apagado")
	}
	if !dbClosed.Load() {
		t.Error("al cerrarse debe liberar la conexión a la base de datos")
	}
}

func TestServe_ApagadoNoEsperaMasDelLimite(t *testing.T) {
	if ShutdownTimeout != 10*time.Second {
		t.Errorf("el límite de apagado debe ser 10 s, es %v", ShutdownTimeout)
	}
	s := NewServer(quietLogger())
	s.shutdownTimeout = 200 * time.Millisecond
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	s.HandleFunc("GET /colgada", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	})
	ln, err := s.Listen(freeAddr(t))
	if err != nil {
		t.Fatalf("no se pudo escuchar: %v", err)
	}
	ctx, stop := context.WithCancel(context.Background())
	var dbClosed atomic.Bool
	served := make(chan error, 1)
	go func() { served <- s.Serve(ctx, ln, func() { dbClosed.Store(true) }) }()
	go func() { _, _ = http.Get("http://" + ln.Addr().String() + "/colgada") }()
	<-started

	inicio := time.Now()
	stop()
	select {
	case <-served:
	case <-time.After(3 * time.Second):
		t.Fatal("Serve debía terminar al vencer el límite de apagado")
	}
	if time.Since(inicio) > 2*time.Second {
		t.Errorf("el apagado tardó %v, más que el límite configurado", time.Since(inicio))
	}
	if !dbClosed.Load() {
		t.Error("aun al vencer el límite debe liberar la conexión a la base de datos")
	}
}

// T015: puerto en uso o no válido (FR-006).

func TestListen_PuertoNoValido(t *testing.T) {
	for _, addr := range []string{"abc", ":70000", ":0", ":-1", "localhost"} {
		_, err := NewServer(quietLogger()).Listen(addr)

		if err == nil {
			t.Errorf("%q: se esperaba un error de puerto no válido", addr)
			continue
		}
		if !strings.Contains(err.Error(), "no es válido") {
			t.Errorf("%q: el mensaje debe decir que el puerto no es válido: %q", addr, err.Error())
		}
	}
}

func TestListen_PuertoEnUsoInformaElPuerto(t *testing.T) {
	ocupado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo ocupar un puerto: %v", err)
	}
	defer ocupado.Close()
	_, port, _ := net.SplitHostPort(ocupado.Addr().String())

	_, err = NewServer(quietLogger()).Listen(ocupado.Addr().String())

	if err == nil {
		t.Fatal("se esperaba un error de puerto en uso")
	}
	msg := err.Error()
	if !strings.Contains(msg, port) || !strings.Contains(msg, "en uso") {
		t.Errorf("el mensaje debe indicar que el puerto %s está en uso: %q", port, msg)
	}
	if strings.Contains(msg, "bind") || strings.Contains(msg, "listen tcp") {
		t.Errorf("el mensaje no debe incluir detalles internos del sistema: %q", msg)
	}
}

// freeAddr devuelve una dirección local con un puerto libre (Listen rechaza el puerto 0).
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo obtener un puerto libre: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func esperar(limite time.Duration, cond func() bool) bool {
	fin := time.Now().Add(limite)
	for time.Now().Before(fin) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}
