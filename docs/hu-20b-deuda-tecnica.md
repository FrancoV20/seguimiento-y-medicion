# HU-20b: deuda técnica, decisiones de alcance y notas de proceso

## Decisiones de alcance
1. **Sin contraseña.** Identificarse es elegir un nombre entre los integrantes del proyecto. No es un mecanismo de seguridad (Assumptions de spec.md).
   Estado: decisión del equipo (2026-10-21): se difiere. Se termina el Sprint 2 sin contraseña y se corrige después como historia nueva.
2. **Sin roles.** No se distingue Scrum Master de Product Builder porque HU-01 no guarda el rol. Distinguirlos exige una migración (columna de rol en `project_members`) y escenarios de permisos en HU-02, HU-04, HU-05, HU-15 y otras.
   Estado: decisión del equipo (2026-10-21): se difiere. Se termina el Sprint 2 sin roles y se corrige después como historia nueva.
3. **Cookies sin firmar.** Se validan contra la base en cada solicitud, así que una cookie manipulada no da acceso a nada que no esté ya abierto. La cookie `flash` solo acepta tres tipos y hasta 300 caracteres, y se muestra siempre escapada: falsificarla solo produce un mensaje falso. Si se exige autenticación real: firmar con HMAC y agregar `SESSION_SECRET` a la configuración.
   Estado: pendiente de tratar con el equipo. Tiene sentido decidirla junto con la historia de contraseña.

## Historias candidatas para el Sprint 3
- Autenticación con contraseña (reemplaza la identificación por nombre).
- Roles y permisos por Scrum Master / Product Builder (migración con el rol y verificaciones en las acciones de cada historia).
- Firma de cookies, si se agrega autenticación.

## Limitaciones conocidas
- `Secure` se agrega solo si la conexión llega por TLS directo (`r.TLS`). Detrás de un proxy habría que leer `X-Forwarded-Proto`.
- Un envío sin integrante identificado vuelve a la pantalla de origen por GET: los datos del formulario no se conservan (research.md, decisión 11).
- "Sesión del navegador" depende del navegador: con "continuar donde lo dejaste" las cookies de sesión sobreviven al cierre. Se verifica con ventana privada.
- Dos pestañas del mismo navegador comparten la selección.
- Cualquier error del almacenamiento se responde como 503, incluso uno que no sea una caída de la base.
- `src/cmd/server` no tiene pruebas automáticas: el cableado de `main.go` se verifica a mano (ver quickstart.md).

## Pendientes opcionales
- T027 y T028 (límite de cuerpo con 413 y cabeceras de seguridad): extras, fuera de los criterios de aceptación.
- Los archivos del repositorio tienen finales de línea CRLF en Windows, y `gofmt -l` los marca todos. Proponer al equipo un `.gitattributes` con `* text=auto eol=lf`.
- La prueba de integración de HU-01 hace `TRUNCATE ... CASCADE` sobre la base de desarrollo y borra los datos locales de cada integrante. Proponer aislarla en un esquema temporal, como hace la de HU-20a.

## Desvíos de proceso (para el acta de la retrospectiva)
- T009 y T010 se commitearon juntos (`9b6dcc6`): el RED falló por un error de copiado, no de comportamiento.
- `security_test.go` se versionó junto con otro commit antes de verlo fallar (T022).
- Varios archivos nuevos se copiaron con una línea de más antes de `package`; pasó un RED por error de sintaxis y no de comportamiento.
- `go build` y `go test` se quedaron sin memoria en Windows (linker). Mitigación: `-ldflags="-s -w"` y limitar la memoria de WSL con `.wslconfig`.
- El puerto 8080 está ocupado por otro proyecto con Docker: se usa `ADDR=:8081`.