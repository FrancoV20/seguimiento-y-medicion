package main

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// T019 (E8, FR-010, FR-011): con una COOKIE_SECRET corta el servidor no arranca, explica el
// largo mínimo, no muestra la clave y no abre el puerto.
func TestRun_ClaveCortaNoArranca(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo reservar un puerto: %v", err)
	}
	addr := probe.Addr().String()
	_ = probe.Close()

	const short = "clave-corta-123"
	env := map[string]string{
		web.EnvDatabaseURL:  "postgres://usuario:clave@127.0.0.1:1/base?sslmode=disable",
		web.EnvAddr:         addr,
		web.EnvCookieSecret: short,
	}
	var stderr bytes.Buffer
	code := run(context.Background(), func(k string) string { return env[k] }, &stderr)

	if code != 1 {
		t.Fatalf("se esperaba código 1, se obtuvo %d", code)
	}
	out := stderr.String()
	if !strings.Contains(out, "32") || !strings.Contains(out, web.EnvCookieSecret) {
		t.Errorf("el mensaje debe explicar el largo mínimo de %s: %q", web.EnvCookieSecret, out)
	}
	if strings.Contains(out, short) {
		t.Errorf("el mensaje no debe mostrar la clave: %q", out)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Errorf("el puerto %s no debía quedar abierto: %v", addr, err)
	} else {
		_ = ln.Close()
	}
}
