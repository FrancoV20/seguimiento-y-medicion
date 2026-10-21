# Research: Formularios seguros y contexto de trabajo

## Decision 1: Post/Redirect/Get y mensajes de una sola vista

- **Decision**: Los envíos válidos redirigen con 303. El mensaje se guarda en una cookie
  `flash` de corta duración y se borra al leerlo. Solo el servidor escribe su contenido, y se
  renderiza escapado.
- **Rationale**: Evita reenvíos al recargar y no necesita tabla ni migración.
- **Alternatives considered**: Mensajes en la URL (descartada: se pueden falsificar y repetir al
  recargar); sesión en BD (descartada: exige migración).

## Decision 2: Formato de mensajes y errores de campo

- **Decision**: Cada tipo de mensaje (éxito, error, advertencia) tiene una clase CSS propia y una
  etiqueta de texto ("Éxito", "Error", "Advertencia") además del color. Los errores de campo se
  renderizan con el parcial `field.html` junto al campo, con los valores enviados conservados. El
  texto "La operación no pudo completarse" es una constante única (`MsgOperationFailed`).
- **Rationale**: La etiqueta de texto evita depender solo del color y deja un único punto de
  cambio para HU-21 a HU-25.
- **Decisión relacionada**: el texto de los mensajes de negocio lo define cada historia de
  dominio; esta historia solo define su formato y el texto único de fallo.

## Decision 3: Protección contra envíos de otro origen

- **Decision**: Envolver el servidor con `http.CrossOriginProtection` de la biblioteca estándar
  (disponible desde Go 1.25), que rechaza solicitudes no seguras entre orígenes usando
  `Sec-Fetch-Site` y `Origin`, con un manejador de rechazo que devuelve 403 con la página de
  error (`RenderError`) y un mensaje claro. Confirmar la API exacta en Go 1.27.1 con
  `go doc net/http.CrossOriginProtection` (T004).
- **Rationale**: Sin tokens en cada formulario, y los formularios de HU-21 a HU-25 la heredan.
  Las solicitudes sin esos encabezados (curl, pruebas) pasan, lo que permite probar sin ruido.
- **Alternatives considered**: Token CSRF por formulario (descartada por ahora: más código en
  cada plantilla); posponer (descartada por el equipo: la robustez pesa en la evaluación).
- **Confirmado (T004)** con `go doc`: `NewCrossOriginProtection()`, `SetDenyHandler(h)` y
  `Handler(h)`. GET/HEAD/OPTIONS siempre pasan; sin `Sec-Fetch-Site` ni `Origin` se permite.

## Decision 4 (opcional): Cabeceras de seguridad y límite de cuerpo

- **Decision**: Middleware que agrega `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: same-origin` y `Content-Security-Policy: default-src 'self'`, y limita el
  cuerpo de formularios a 1 MiB con `http.MaxBytesReader` (responde 413).
- **Rationale**: Como todos los recursos son locales, la política de contenido es posible y
  refuerza el escape de HTML. No es un criterio de aceptación: se implementa solo si sobra tiempo
  (T027 y T028); si el equipo lo quiere obligatorio, hay que agregar los escenarios.

## Decision 5: Cookies de sesión sin tabla ni migración

- **Decision**: Guardar el proyecto y el integrante activos en dos cookies de sesión (`ctx_proyecto`
  y `ctx_integrante`), sin `Expires`, con `HttpOnly`, `SameSite=Lax` y `Secure` cuando hay HTTPS.
- **Rationale**: Es la decisión del equipo. No necesita tabla nueva ni migración (evita los
  conflictos de numeración y orden de migraciones vistos en el Sprint 1), se limpia al cerrar el
  navegador y es lo más simple para un sistema sin inicio de sesión.
- **Alternatives considered**: Cookie persistente (descartada: deja estado viejo y expone la
  identidad en equipos compartidos); sesión en base de datos (descartada: exige migración y
  es desproporcionada sin autenticación).

## Decision 6: Las cookies no se firman; se validan contra la base en cada solicitud

- **Decision**: No firmar las cookies. En cada solicitud con cookies, validar que el proyecto
  existe y que el integrante pertenece a ese proyecto, con una sola consulta.
- **Rationale**: La identificación no es un mecanismo de seguridad (no hay contraseña), por lo que
  firmarla no aporta protección real y agregaría una variable de configuración más al arranque.
  Validar contra la base cubre cookies manipuladas, proyectos eliminados e integrantes quitados.
  Es una decisión de alcance a confirmar con la cátedra.
- **Alternatives considered**: Firmar con HMAC (descartada por ahora: agrega `SESSION_SECRET` sin
  proteger nada que no esté ya abierto); confiar en la cookie sin validar (descartada: permite
  acciones con integrantes ajenos).

## Decision 7: Sin roles en esta historia

- **Decision**: No distinguir Scrum Master de Product Builder en la interfaz; se registra como
  deuda técnica.
- **Rationale**: HU-01 no guarda el rol de cada integrante; distinguirlos exigiría cambiar el
  modelo de datos y agregar escenarios de permisos en varias historias del sprint.
- **Alternatives considered**: Distinguir roles ahora (descartada: amplía el alcance y toca
  tablas ya entregadas).

## Decision 8: Sin autoselección de proyecto

- **Decision**: Aunque haya un solo proyecto, siempre se pide elegirlo.
- **Rationale**: Un único comportamiento, predecible y simple de probar, al costo de una
  interacción extra.
- **Alternatives considered**: Autoseleccionar cuando hay uno (descartada: comportamiento que
  cambia al crear el segundo proyecto y suma casos de prueba).

## Decision 9: Una sola consulta para resolver el contexto

- **Decision**: `ContextStore.ResolveContext(ctx, projectID, memberID)` devuelve en una consulta si
  el proyecto existe (con su nombre) y si el integrante pertenece a él (con su nombre). Además,
  `ListProjects` y `ListMembers(projectID)` alimentan los selectores.
- **Rationale**: Mantiene la validación por solicitud por debajo de 50 ms y concentra la lectura
  en un adaptador testeable con PostgreSQL real, como exige el principio VI.
- **Alternatives considered**: Dos consultas separadas (descartada: más viajes a la base);
  reutilizar el store de HU-01 (a evaluar en T001: solo si ya expone estas lecturas; si no, se
  agrega esta interfaz de solo lectura).

## Decision 10: Limpieza con advertencia, no con error

- **Decision**: Si el proyecto o el integrante dejan de ser válidos, el middleware borra la cookie
  correspondiente, escribe una advertencia con `SetFlash` y deja seguir la solicitud como "sin
  selección". Las secciones protegidas redirigen entonces a la selección.
- **Rationale**: El usuario entiende por qué se le pide elegir de nuevo y no ve un error técnico.
- **Decisión relacionada**: si la validación falla por indisponibilidad de la base de datos, no se
  limpia nada y se responde 503 vía `Fail`, para no perder la selección por una caída transitoria.

## Decision 11: Protección de secciones con destino de regreso seguro

- **Decision**: `RequireContext` redirige con 303 a `/contexto/proyecto` o `/contexto/integrante`
  con el parámetro `volver_a`. El destino se acepta solo si empieza con una sola `/`, no contiene
  `//`, `\`, esquema ni caracteres de control, y su primer segmento corresponde a una sección
  registrada. En caso contrario se usa `/`.
- **Rationale**: Evita redirecciones abiertas y destinos arbitrarios, y cumple el escenario de
  volver a la sección pedida.
- **Alternatives considered**: Guardar el destino en una cookie (descartada: más estado y
  confusión con varias pestañas); aceptar cualquier ruta local (descartada: permite rutas no
  registradas).
- **Limitación aceptada**: si se pierde la identificación al enviar un formulario, los datos del
  formulario no se conservan; se vuelve a la pantalla de origen por GET.

## Decision 12: Cambio de proyecto limpia el integrante

- **Decision**: Al seleccionar un proyecto distinto del activo se borra la cookie de integrante; al
  seleccionar el mismo se conserva.
- **Rationale**: El integrante pertenece al proyecto anterior; conservarlo produciría un
  contexto inconsistente.

## Decision 13: Secciones de prueba hasta que existan las reales

- **Decision**: Los escenarios sobre "una sección que requiere proyecto o integrante" se verifican
  con secciones de prueba registradas solo en las pruebas (una sin requisitos, una con
  `RequiresProject` y una con `RequiresMember`).
- **Rationale**: Las secciones reales (Backlog, Sprints, Esfuerzo, Métricas) llegan en HU-22 a
  HU-25, después de esta historia.

## Decision 14: Estrategia de pruebas

- **Decision**: Pruebas unitarias del saneamiento de `volver_a`, de la lectura de cookies y del
  flash; pruebas de handler con `httptest` y `cookiejar` que ejecutan el flujo completo sobre el
  servidor y el `PostgreSQLContextStore` reales; integración del adaptador con fixtures de dos
  proyectos y varios integrantes. Se corre con `go test -count=1 -p 1 ./...`.
- **Rationale**: Verifica el comportamiento real, incluyendo proyectos eliminados e integrantes
  quitados, sin mocks que sustituyan las reglas.
- **Alternatives considered**: Store en memoria para los handlers (descartada: no verifica las
  consultas ni el principio VI).
