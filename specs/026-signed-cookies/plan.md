# Plan de implementación: HU-26 – Cookies firmadas

**Rama**: feature/hu-26-cookies-firmadas | **Spec**: specs/026-signed-cookies/spec.md
**Constitución**: v1.1.0

## Resumen

Se agrega un firmador de cookies (`CookieSigner`) y la carga de la clave desde
`COOKIE_SECRET` en `src/web`. El firmador se aplica en los dos únicos puntos
donde HU-20b lee y escribe las cookies de proyecto e integrante:
`readIDCookie` y `setContextCookie` (`src/web/context.go`). `main.go` carga la
clave al arrancar. No hay base de datos, migraciones ni puertos nuevos.

## Estado verificado en `main` (27/10/2026)

- `src/web/context.go` define `CookieProject = "ctx_proyecto"`,
  `CookieMember = "ctx_integrante"`, `readIDCookie(r, name) (id, present)`,
  `newContextCookie`, `setContextCookie(w, r, name, id)`, `clearContextCookie`
  y `parseID`.
- `contextMiddleware` (`context_middleware.go`) trata `id == 0` con la cookie
  presente como cookie manipulada: la borra en silencio. Sin proyecto válido
  descarta ambas cookies.
- `context_handlers.go` llama a `setContextCookie` al elegir proyecto e
  integrante.
- `flash.go` usa `newContextCookie` para la cookie `flash` (sin firmar, a
  propósito).
- `src/cmd/server/main.go` ejecuta `LoadConfig` y después `NewServer`, `Listen`
  y `OpenDatabase`.

## Contexto técnico

- Go 1.27.1, solo biblioteca estándar: `crypto/hmac`, `crypto/sha256`,
  `crypto/rand`, `encoding/base64`, `sync/atomic`.
- Pruebas: unitarias en `src/web`, de aplicación en `test/` (con PostgreSQL,
  igual que las demás) y una prueba de `run` en `src/cmd/server`.
- Sin dependencias nuevas ni migraciones.

## Verificación de la constitución

| Principio | Cumplimiento |
|---|---|
| SDD → BDD → TDD | Escenarios con pruebas rojas primero |
| Trazabilidad | Tabla en spec.md |
| Persistencia PostgreSQL (VI) | No aplica: no agrega persistencia ni puertos de datos |
| Seguridad | Tiempo constante, clave nunca registrada, falla temprana con clave débil |
| Otras historias | Se editan solo `context.go`, `main.go` y los helpers y pruebas que arman cookies a mano |

## Estructura de archivos

Nuevos:

- `src/web/cookie_signer.go` – `CookieSigner` con `Sign(name, value)` y
  `Verify(name, signed)`; clave global del proceso; `SetCookieSecret`;
  `SignedContextValue(name, id)`.
- `src/web/cookie_secret.go` – `EnvCookieSecret`, `MinCookieSecretLength`,
  `ShortCookieSecretError`, `LoadCookieSecret`.
- `src/web/cookie_signer_test.go` y `src/web/cookie_secret_test.go`.
- `test/web_signed_cookies_bdd_test.go`.
- `src/cmd/server/main_test.go` (prueba de clave corta).

Editados:

- `src/web/context.go` – `readIDCookie` verifica la firma y `setContextCookie` la
  agrega.
- `src/cmd/server/main.go` – carga y fija la clave antes de abrir el puerto.
- Pruebas existentes que arman o leen el valor de la cookie (lista en T001):
  `src/web/context_test.go`, `context_handlers_test.go`, `context_member_test.go`,
  `projects_handlers_test.go` y `test/context_bdd_test.go`.
- `README.md` – una nota sobre `COOKIE_SECRET` junto a `DATABASE_URL`.

## Decisiones de diseño

1. **Clave global del proceso** (`atomic.Pointer`). Se fija una vez al arrancar.
   Evita cambiar las firmas de `readIDCookie`, `setContextCookie` y
   `clearContextCookie` y de las decenas de pruebas que las usan.
2. **Clave temporal por defecto.** Al cargar el paquete se genera una clave
   aleatoria. Así `NewServer` y todas las pruebas siguen funcionando sin
   configurar nada.
3. **Formato** `<id>.<firma>`; se separa por el último punto; firma HMAC-SHA256
   en base64 URL sin relleno sobre `nombre 0x00 id`.
4. **Descarte con el camino existente.** `readIDCookie` devuelve
   `(0, true)` ante una firma inválida. El middleware de HU-20b ya borra en
   silencio las cookies presentes con id 0.
5. **`LoadCookieSecret(getenv)`** devuelve `(secret, temporary, err)`, para
   probarla sin tocar el entorno real. `main` la llama justo después de
   `LoadConfig`.
6. **No se firma `flash`.** Su texto lo define siempre el servidor y se escapa.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Pruebas de ramas abiertas que arman cookies con un número suelto | Helpers `idCookie` y `cookieID` firman; aviso en las notas del Issue |
| Pruebas que cambian la clave global | No se usa `t.Parallel` en `src/web`; las pruebas restauran la clave con `t.Cleanup` |
| Conflicto de merge en `context.go` | El cambio toca dos funciones; coordinar con quien edite ese archivo |