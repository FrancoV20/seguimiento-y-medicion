# Especificación: HU-26 – Cookies firmadas de contexto de trabajo

**Rama**: feature/hu-26-cookies-firmadas
**Estado**: Propuesta (historia nueva, pendiente de aceptación del equipo)
**Constitución**: v1.1.0
**Depende de**: HU-20b (en `main`)

## Historia de usuario

Como Product Builder, quiero que el proyecto y el integrante activos viajen en
cookies firmadas, para que nadie pueda falsificar el contexto de trabajo
modificando su navegador.

## Sección SDD

### Intención

Que el servidor solo confíe en un contexto de trabajo (proyecto e integrante)
que él mismo emitió. Hoy las cookies `ctx_proyecto` y `ctx_integrante` guardan
el identificador en texto plano: cualquiera puede escribir otro número, por
ejemplo el de un integrante que no pertenece al proyecto. HU-26 no agrega
autenticación: sin contraseña, cualquiera puede seguir eligiendo cualquier
proyecto desde la pantalla de selección.

### Alcance

- Firma al emitir y verificación al leer las cookies `ctx_proyecto` y
  `ctx_integrante`.
- Configuración de la clave secreta por la variable de entorno `COOKIE_SECRET`.
- Descarte de cookies sin firma o con firma inválida (incluidas las emitidas
  antes de esta mejora).

### Fuera de alcance

- Contraseñas, inicio de sesión y roles.
- Firma de la cookie `flash` (el texto lo define el servidor y se muestra
  escapado; no autoriza ninguna acción).
- Vencimiento firmado y rotación de claves.
- Cifrado del contenido (el valor sigue siendo legible; solo se protege su
  integridad).
- Cambios en los atributos de las cookies (ya definidos por HU-20b).

### Contratos

- Valor de la cookie: `<id>.<firma>`. La firma es HMAC-SHA256, en base64 URL sin
  relleno, calculada sobre `<nombre de la cookie> 0x00 <id>`. Incluir el nombre
  impide copiar el valor firmado de una cookie a otra.
- La verificación usa comparación en tiempo constante.
- Una cookie sin firma, con firma incorrecta o mal formada se comporta como una
  cookie manipulada de HU-20b: se descarta en silencio y se borra del navegador.
- Configuración: `COOKIE_SECRET`, mínimo 32 caracteres (bytes ASCII). Ausente:
  clave temporal aleatoria con aviso. Menor al mínimo: la aplicación no arranca.
- No cambia ninguna función exportada existente. Se agregan `SetCookieSecret`,
  `LoadCookieSecret` y `SignedContextValue` (esta última, para las pruebas de la
  carpeta `test`).

## Escenarios BDD

Los 8 escenarios del Issue HU-26 (texto idéntico):

1. Un contexto válido se conserva.
2. Cookie de proyecto modificada.
3. Cookie de integrante modificada.
4. Valor firmado copiado a otra cookie.
5. Cookies sin firma de una versión anterior.
6. Clave secreta configurada.
7. Clave secreta ausente.
8. Clave secreta demasiado corta.

## Requisitos funcionales

- **FR-001**: Las cookies `ctx_proyecto` y `ctx_integrante` MUST emitirse con
  valor `<id>.<firma>`.
- **FR-002**: Al leer una cookie de contexto, el sistema MUST verificar la firma
  y usar el identificador solo si la firma es válida.
- **FR-003**: Una cookie sin firma, vacía, mal formada o con firma incorrecta
  MUST tratarse como una cookie manipulada: se descarta en silencio, se borra del
  navegador y la solicitud sigue como si no existiera, sin error técnico visible.
- **FR-004**: Una cookie de proyecto inválida MUST descartar también la cookie
  de integrante (no hay integrante sin proyecto; regla vigente de HU-20b).
- **FR-005**: Una cookie de integrante inválida MUST descartarse sin afectar al
  proyecto activo; las secciones que requieren integrante MUST pedir
  identificarse con el mensaje vigente de HU-20b.
- **FR-006**: La firma MUST depender del nombre de la cookie, de modo que el
  valor firmado de una cookie no sea válido en la otra.
- **FR-007**: La comparación de firmas MUST hacerse en tiempo constante.
- **FR-008**: Al arrancar con `COOKIE_SECRET` de 32 caracteres o más, el sistema
  MUST usarla como clave; las cookies emitidas con la misma clave MUST seguir
  siendo válidas después de reiniciar.
- **FR-009**: Si `COOKIE_SECRET` no está definida, el sistema MUST generar una
  clave aleatoria de 32 bytes, registrar un aviso de que las sesiones se
  perderán al reiniciar y funcionar con normalidad.
- **FR-010**: Si `COOKIE_SECRET` está definida con menos de 32 caracteres, la
  aplicación MUST NOT arrancar y MUST informar el largo mínimo, sin imprimir el
  valor configurado. La comprobación MUST ocurrir antes de abrir el puerto o la
  base de datos.
- **FR-011**: La clave MUST NOT aparecer en registros, respuestas ni mensajes de
  error.
- **FR-012**: Los atributos de las cookies definidos por HU-20b (`HttpOnly`,
  `SameSite=Lax`, `Path=/`, sin `Expires`, `Secure` por TLS) MUST conservarse
  sin cambios.
- **FR-013**: Un usuario que navega normalmente (elige proyecto, se identifica y
  usa las secciones) MUST ver el mismo comportamiento que antes.

## Entidades clave

- **Cookie de contexto**: `ctx_proyecto` y `ctx_integrante`.
- **Clave secreta**: configuración de la aplicación, no se guarda en la base de
  datos.

## Criterios de éxito

- **SC-001**: Ninguna cookie de contexto modificada, copiada o sin firma es
  aceptada en las pruebas.
- **SC-002**: La navegación normal funciona igual que antes (FR-013).
- **SC-003**: Las cookies emitidas con la misma clave siguen siendo válidas tras
  un reinicio simulado y dejan de serlo con otra clave.
- **SC-004**: Todas las pruebas existentes siguen en verde.
- **SC-005**: Cada escenario tiene una prueba que falló antes del código
  (evidencia RED) y pasa después (evidencia GREEN), y la cobertura queda en
  `cobertura.txt`.

## Supuestos y decisiones

- La clave es global del proceso (se fija al arrancar), porque las funciones de
  cookies de HU-20b no reciben el servidor. Ver `research.md`.
- La firma no incluye vencimiento: la cookie sigue siendo de sesión.
- El largo de la clave se mide en bytes después de recortar espacios en los
  bordes, igual que `LoadConfig`.

## Trazabilidad

| Historia | Escenario | Requisitos | Éxito | Prueba BDD | Prueba unitaria |
|---|---|---|---|---|---|
| HU-26 | 1 Contexto válido | FR-001, FR-002, FR-013 | SC-002, SC-004 | `TestHU26_ContextoValidoSeConserva` | `TestCookieSigner_RoundTrip` |
| HU-26 | 2 Proyecto modificado | FR-002, FR-003, FR-004 | SC-001 | `TestHU26_CookieProyectoModificada` | `TestCookieSigner_ValorModificado` |
| HU-26 | 3 Integrante modificado | FR-002, FR-003, FR-005 | SC-001 | `TestHU26_CookieIntegranteModificada` | `TestCookieSigner_FirmaModificada` |
| HU-26 | 4 Valor copiado | FR-006 | SC-001 | `TestHU26_ValorCopiadoEntreCookies` | `TestCookieSigner_LigadoAlNombre` |
| HU-26 | 5 Sin firma anterior | FR-003 | SC-001 | `TestHU26_CookiesSinFirmaSeDescartan` | `TestCookieSigner_SinFirma`, `TestCookieSigner_MalFormada` |
| HU-26 | 6 Clave configurada | FR-008, FR-011 | SC-003 | `TestHU26_ReinicioConMismaClave` | `TestLoadCookieSecret_Valida` |
| HU-26 | 7 Clave ausente | FR-009, FR-011 | SC-002 | — | `TestLoadCookieSecret_Ausente` |
| HU-26 | 8 Clave corta | FR-010, FR-011 | SC-001 | `TestRun_ClaveCortaNoArranca` (en `src/cmd/server`) | `TestLoadCookieSecret_Corta` |
| HU-26 | Regresión de atributos | FR-007, FR-012 | SC-004 | — | `TestSetContextCookie_AtributosDeSesion` (existente, actualizada) |