# Research: Base de la aplicación web

## Decision 1: SSR con `html/template` y recursos embebidos

- **Decision**: Renderizar HTML en el servidor con `html/template` y embeber plantillas y CSS con
  `embed`. Sin JavaScript ni librerías externas en esta historia.
- **Rationale**: Es el enfoque elegido por el equipo para el frontend. Los recursos embebidos
  funcionan sin internet y sin servir archivos sueltos, y las plantillas se parecen al HTML que el
  equipo ya conoce.
- **Alternatives considered**: SPA con API JSON (descartada: duplica contratos y el equipo eligió
  SSR); CDN para estilos (descartada: no funciona sin conexión).

## Decision 2: Registro de secciones que alimenta el menú

- **Decision**: Cada historia registra `Section{Name, Path, RequiresProject, RequiresMember}` y el
  menú se arma solo con las registradas. Una dirección duplicada hace fallar el arranque.
- **Rationale**: Resuelve "secciones no implementadas ocultas" sin banderas manuales que se puedan
  olvidar, y HU-20b reutiliza los indicadores para exigir contexto.
- **Alternatives considered**: Lista fija con secciones deshabilitadas (descartada: hay que
  mantenerla a mano y habilitarla en cada historia).

## Decision 3: Configuración por variables de entorno con redacción

- **Decision**: Leer `DATABASE_URL` (obligatoria), `ADDR` (opcional, por defecto `:8080`) y los
  tiempos como constantes. Los errores nombran la variable, nunca su valor; al registrar la URL se
  usa `url.URL.Redacted()`.
- **Rationale**: Cumple "informa qué falta sin mostrar valores sensibles" y evita filtrar la clave
  en logs.
- **Alternatives considered**: Archivo de configuración (descartada: agrega un archivo más con
  secretos en el repositorio).

## Decision 4: Reintentos de conexión limitados

- **Decision**: Hasta 10 intentos con 2 s de espera (aprox. 20 s) usando `PingContext`; luego
  falla con mensaje sin credenciales ni URL completa.
- **Rationale**: Con `docker compose` PostgreSQL tarda unos segundos en aceptar conexiones.
- **Alternatives considered**: Sin reintentos (descartada: arranques fallidos intermitentes);
  reintentos indefinidos (descartada: oculta fallas reales).

## Decision 5: El servidor falla si el esquema no está al día

- **Decision**: Al iniciar, leer `schema_migrations` (versión y `dirty`) y compararla con la mayor
  versión de los archivos `.up.sql` embebidos. Versión menor o `dirty` impide iniciar con mensaje
  accionable; versión mayor inicia con advertencia en el registro.
- **Rationale**: Falla temprano y claro en vez de errores "relation does not exist" a mitad de
  uso; el README ya pide avisar al equipo antes de usar `force`.
- **Alternatives considered**: Solo advertir (descartada: deja la aplicación a medias).
- **Riesgo conocido**: `migrate up` solo aplica archivos con número mayor a la versión actual. Si
  una migración con número menor se integra después de que la base ya aplicó una mayor, no se
  aplica y esta verificación no lo detecta, porque `schema_migrations` guarda una sola versión.
  Fue lo que pasó con la migración 003 en el Sprint 1.
  **Mitigación**: la regla del equipo (una migración nueva toma el último número de `main` más
  uno), una prueba de integración que aplica todos los `.up.sql` en orden sobre una base
  descartable (T043) y recrear la base local con `docker compose down -v` al integrar ramas.

## Decision 6: Traducción de errores y página 503

- **Decision**: Los errores de aplicación se traducen en un solo lugar (`Fail`): no encontrado →
  404, base de datos no disponible (error de conexión, `driver.ErrBadConn`,
  `context.DeadlineExceeded`) → 503, cualquier otro → 500 genérico. El detalle se registra con
  `slog`. Para cualquier otro estado (405 y el 403 de HU-20b) existe `RenderError`.
- **Rationale**: `database/sql` reabre conexiones solo, por lo que no hace falta reiniciar para
  recuperarse; basta con no cachear el estado de falla. La estructura común no consulta la base,
  por eso el 503 se muestra completo.
- **Alternatives considered**: Estado global "base caída" con reintento en segundo plano
  (descartada: más piezas y riesgo de quedar bloqueado).

## Decision 7: Recuperación de pánicos

- **Decision**: Middleware `recover` que registra el pánico con traza en el log, responde la
  página 500 genérica y deja el servidor en pie.
- **Rationale**: `net/http` ya aísla pánicos por conexión, pero responde sin página y con ruido;
  el middleware garantiza la página de error y el registro.

## Decision 8: Render a buffer

- **Decision**: Ejecutar la plantilla sobre un `bytes.Buffer` y escribir la respuesta solo si
  terminó sin error.
- **Rationale**: Cumple FR-017: una falla de plantilla produce la página de error completa y no
  HTML a medias.

## Decision 9: Apagado ordenado

- **Decision**: `signal.NotifyContext` para interrupciones y `Server.Shutdown` con 10 s de
  tiempo máximo; después se cierra el pool de la base de datos.
- **Rationale**: Completa solicitudes en curso y evita cortes en medio de un guardado.

## Decision 10: Páginas 404 y 405 con la estructura común

- **Decision**: Envolver el `ServeMux` con un manejador que, cuando el mux no encuentra una ruta,
  decide 404 o 405 (según el método y el encabezado `Allow`) y renderiza la página de error común
  con `RenderError`, en lugar del texto plano por defecto. Un parámetro de dirección malformado,
  negativo o muy grande se trata como 404 en el handler que lo recibe, mediante `Fail`.
- **Rationale**: El `ServeMux` de Go responde 404 y 405 en texto plano; los escenarios exigen la
  estructura común y un enlace al inicio.
- **A verificar** (T041): el comportamiento exacto de `mux.Handler(r)` ante rutas con método en
  Go 1.27.1; si no alcanza, registrar las rutas sin método y validar el método en un auxiliar.

## Decision 11: Estrategia de pruebas

- **Decision**: Pruebas de handler con `httptest` sobre el servidor real, BDD en
  `test/web_base_bdd_test.go` y pruebas de integración con PostgreSQL. La caída y recuperación de
  la base de datos se prueba con un proxy TCP local que se pausa y se reanuda delante de
  PostgreSQL. La prueba de todas las migraciones usa una base descartable creada y borrada por la
  propia prueba, nunca la base de desarrollo. Se corre con `go test -count=1 -p 1 ./...`.
- **Rationale**: Verifica el comportamiento real sin mocks que sustituyan las reglas y sin
  depender de detener contenedores desde la prueba. `-p 1` evita que los paquetes de integración
  se pisen al compartir una base.
- **Alternatives considered**: Detener el contenedor desde la prueba (descartada: lenta e
  inestable); cerrar el pool (descartada: no permite probar la recuperación).

## Decision 12 (opcional): Tiempo máximo por solicitud

- **Decision**: Un middleware que limita cada solicitud con un contexto con plazo. No es un
  criterio de aceptación; se implementa solo si sobra tiempo (T042).
- **Rationale**: Evita solicitudes colgadas, pero no está pedido por ningún escenario.
