# Modelo de datos: HU-26 – Cookies firmadas

## Cambios de esquema

Ninguno. No se crean tablas ni migraciones.

## Configuración

| Nombre | Regla |
|---|---|
| `COOKIE_SECRET` | Mínimo 32 caracteres (bytes ASCII, sin espacios en los bordes). Ausente: clave temporal de 32 bytes aleatorios y aviso en el registro. Menor a 32: la aplicación no arranca. |

## Estructuras y funciones (paquete `web`)

- `CookieSigner{key []byte}`
  - `Sign(name, value string) string` →
    `value + "." + base64url(HMAC-SHA256(key, name + "\x00" + value))`
  - `Verify(name, signed string) (value string, ok bool)` → separa por el último
    `.`, recalcula la firma y compara con `hmac.Equal`. Devuelve `ok=false` ante
    firma incorrecta, valor o firma vacíos, o ausencia de punto.
- Clave global del proceso: `atomic.Pointer[CookieSigner]`, inicializada con una
  clave aleatoria al cargar el paquete.
- `SetCookieSecret(secret []byte)`: reemplaza la clave global (se llama una vez
  al arrancar y en las pruebas).
- `SignedContextValue(name string, id int64) string`: valor firmado de una cookie
  de contexto; uso previsto en pruebas.
- `LoadCookieSecret(getenv func(string) string) (secret []byte, temporary bool, err error)`
  - Variable ausente o vacía → 32 bytes de `crypto/rand`, `temporary = true`.
  - 32 bytes o más → esos bytes, `temporary = false`.
  - Menos de 32 → `*ShortCookieSecretError` ("la variable de entorno
    COOKIE_SECRET debe tener al menos 32 caracteres"), sin el valor.

## Cambios en las funciones de HU-20b

| Función | Antes | Después |
|---|---|---|
| `setContextCookie(w, r, name, id)` | valor `"7"` | valor `Sign(name, "7")` |
| `readIDCookie(r, name)` | `parseID(c.Value)` | `Verify(name, c.Value)`; si falla, `(0, true)`; si pasa, `parseID(valor)` |
| `clearContextCookie`, `newContextCookie` | sin cambios | sin cambios |

## Casos de verificación

| Entrada (cookie `ctx_proyecto`) | `readIDCookie` |
|---|---|
| `7.<firma correcta para ctx_proyecto>` | `(7, true)` |
| `8.<firma de "7">` | `(0, true)` |
| `7` (sin firma, formato anterior) | `(0, true)` |
| `""`, `.`, `7.`, `.abc`, `abc` | `(0, true)` |
| Firma de `ctx_proyecto` en `ctx_integrante` | `(0, true)` |
| Firma válida con otra clave | `(0, true)` |
| Firma válida pero valor `abc` o `-1` o `05` | `(0, true)` (lo rechaza `parseID`) |
| Cookie ausente | `(0, false)` |

## Cookies (sin cambios, definidos en HU-20b)

`HttpOnly`, `SameSite=Lax`, `Path=/`, sin `Expires` ni `Max-Age`, `Secure` si la
solicitud llegó por TLS.