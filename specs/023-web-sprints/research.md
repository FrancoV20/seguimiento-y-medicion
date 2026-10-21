# Research: Interfaz web de Sprints

## Decision 1: Una sola pantalla "Sprints"

- **Decision**: `/sprints` muestra el Sprint activo con sus historias y, según el caso, el formulario
  de creación (sin Sprint activo) o el de agregar historias (con Sprint activo). Las acciones son
  `POST` que redirigen con 303 a la misma dirección.
- **Rationale**: Los escenarios hablan de "abrir Sprint activo" y "crear Sprint" sobre la misma
  planificación; una pantalla evita navegación extra y deja resueltos los estados vacíos.
- **Alternatives considered**: Pantallas separadas de creación y de Sprint activo (descartada: agrega
  rutas sin aportar a los criterios).

## Decision 2: Catálogo de lectura propio de HU-23

- **Decision**: Interfaz `SprintCatalog` con `ActiveSprint(ctx, projectID)` y
  `ListStoryOptions(ctx, projectID)`, y su adaptador `PostgreSQLSprintCatalog`.
- **Rationale**: El `SprintStore` de HU-04 tiene `CreateSprint`, `AssignStories` y `RemoveStory`, pero
  ninguna lectura. Un catálogo de solo lectura no toca el núcleo de HU-04 y lo pueden reutilizar HU-16
  y HU-24.
- **Alternatives considered**: Ampliar `SprintStore` (descartada: modifica HU-04 más de lo necesario);
  reutilizar el catálogo de historias de HU-22 (descartada: no conoce asignaciones ni Sprints).

## Decision 3: Un solo Sprint activo, regla del dominio

- **Decision**: La regla la aplica HU-04 (FIX-HU-04, `ErrActiveSprintExists`). La pantalla consulta el
  Sprint activo solo para decidir qué mostrar (el Sprint y no el formulario) y, si igual llega un
  envío, traduce el rechazo a "Ya hay un Sprint activo (Sprint N)" con 409.
- **Rationale**: La lógica de negocio va fuera de la capa web (DoD y constitución II), y hace cierto el
  singular de "el Sprint activo" que usan HU-12, HU-14 y HU-24 aunque alguien llame al servicio
  directamente.
- **Riesgo aceptado**: en una base de desarrollo con un Sprint de prueba no se puede crear otro hasta
  que HU-15 permita cerrarlo; se sale cerrándolo a mano con SQL o recreando la base.
- **Alternatives considered**: Bloquear solo en el handler (descartada: dos copias de la regla);
  restricción en la base (descartada por el equipo, decisión del 20/10/2026).

## Decision 4: Disponibilidad de historias

- **Decision**: Una historia se puede marcar si está en estado `'Pendiente'` y no tiene asignación a
  ningún Sprint. Las demás se muestran deshabilitadas con el motivo: "Ya está asignada al Sprint N" o
  "No está pendiente".
- **Rationale**: El estado `'En Sprint'` que agrega la migración `004` deja de ser "Pendiente", así
  que "disponible" es "pendiente y sin asignación". Mostrar las no disponibles con motivo sigue el
  criterio original de la historia y evita que alguien crea que faltan historias.
- **Alternatives considered**: Ocultar las no disponibles (descartada: contradice el criterio de
  "aparece deshabilitada con el motivo").

## Decision 5: Estado "En Sprint" y consecuencias para otras historias

- **Decision**: Al asignar, HU-04 deja la historia en `'En Sprint'`; al quitarla, vuelve a
  `'Pendiente'`. HU-23 muestra ese estado tal cual.
- **Consecuencias**: El equipo votó cuatro estados (Pendiente, En Sprint, En progreso, Hecho) el
  20/10/2026. HU-14 debe partir de `'En Sprint'` hacia `'En progreso'` y `'Hecho'`; el texto de HU-14,
  HU-08, HU-13 y HU-15 se corrige en cada historia.

## Decision 6: Fuente de las asignaciones

- **Decision**: El catálogo lee las asignaciones de `sprint_story_assignments` (única por historia) y
  el estado de `user_stories`. Se verifica en T001 que HU-04 mantiene `user_stories.active_sprint_id`
  consistente con esa tabla.
- **Rationale**: La unicidad por historia garantiza una sola asignación; leer de una fuente evita
  discrepancias si `active_sprint_id` quedara desactualizado.
- **Nota para HU-15**: como `sprint_story_assignments` es única por historia, el cierre del Sprint
  debe eliminar la asignación de las historias no terminadas para que puedan asignarse a otro Sprint.

## Decision 7: No duplicar las reglas de HU-04

- **Decision**: El handler solo arma el envío (fechas, identificadores sin repetidos, numéricos) y
  traduce los errores de HU-04 a campos. El Goal obligatorio, el período válido, la elegibilidad, la
  exclusividad y el Sprint activo único los decide HU-04.
- **Rationale**: Una única fuente de reglas y mensajes. HU-04 rechaza identificadores repetidos como
  "historia no disponible", por eso el handler los quita antes de llamar.
- **Excepción**: la regla de quitar es nueva y vive en el dominio (Decision 11), no en el handler.
- **Alternatives considered**: Revalidar en el handler (descartada: dos copias que pueden divergir).

## Decision 8: Proyecto y Sprint del contexto del servidor

- **Decision**: La sección se registra con `RequiresProject: true`; el proyecto sale de
  `ActiveContextFrom(r.Context())` y el Sprint activo se obtiene por consulta. Cualquier identificador
  de proyecto o de Sprint del formulario se ignora, y una historia de otro proyecto se rechaza.
- **Rationale**: Evita operar sobre otro proyecto con un envío manipulado y reutiliza la redirección de
  HU-20b cuando no hay proyecto.

## Decision 9: Asignación de todo o nada

- **Decision**: Tanto al crear como al agregar, si una historia no está disponible no se asigna
  ninguna y se informa cuáles.
- **Rationale**: Es el comportamiento de HU-04 (sin asociaciones parciales) y evita estados
  ambiguos para quien planifica.

## Decision 10: Agregar historias al Sprint activo

- **Decision**: Un formulario "Agregar historias" dentro del Sprint activo asigna las historias
  disponibles marcadas con `AssignStories`.
- **Rationale**: HU-04 permite crear un Sprint sin historias y esta pantalla informa la advertencia;
  sin una forma de agregarlas, un Sprint vacío no tendría salida desde la web.
- **Alternatives considered**: No ofrecerlo (descartada: deja un callejón sin salida).

## Decision 11: Quitar historias con la regla en el dominio

- **Decision**: Se puede quitar solo una historia en `'En Sprint'` y sin horas registradas. La
  validación se agrega a `RemoveStory` en `src/sprint` (T047) con una función pura `CanRemoveStory`
  que también usa la pantalla para deshabilitar el botón con el motivo. "Quitar" forma parte de HU-23
  (decisión del equipo): no es una historia aparte.
- **Rationale**: `RemoveStory` de HU-04 no valida nada: quitaría una historia en progreso o hecha y
  dejaría horas huérfanas (HU-03). Una regla solo en el handler dejaría el dominio sin protección.
- **Carrera aceptada**: entre la lectura de la vista y el quitar alguien podría cargar horas; el
  dominio vuelve a comprobarlas al quitar y la carrera restante se documenta.
- **Alternatives considered**: Regla solo en el handler (descartada: lógica de negocio en la capa
  web); quitar cualquier historia (descartada: horas huérfanas); historia aparte (descartada por el
  equipo).

## Decision 12: Códigos de respuesta

- **Decision**: Envío válido: 303 a `/sprints`. Error de validación: 422 con el formulario y los
  datos. Conflicto (ya hay un Sprint activo, historias no disponibles o historia no removible): 409.
  Falla de guardado: 500 con `MsgOperationFailed`. Falla de lectura: `Fail` de HU-20a. Sin proyecto
  activo: 303 de HU-20b.
- **Rationale**: Distingue los casos para las pruebas y el navegador sin cambiar lo que ve la persona.

## Decision 13: Estrategia de pruebas

- **Decision**: Pruebas unitarias del armado del formulario y de la regla de quitar (en el dominio);
  pruebas de handler con `httptest` y `cookiejar` contra el servicio de HU-04 y PostgreSQL reales,
  eligiendo el proyecto activo por el flujo de HU-20b; un almacenamiento que falla solo para simular
  la falla de guardado y de lectura; integración de `PostgreSQLSprintCatalog` con fixtures propias; y
  escenarios BDD.
- **Rationale**: Verifica el comportamiento real, incluido el aislamiento entre proyectos, la
  exclusividad y la redirección sin contexto.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada (los Sprints y sus
  asignaciones sí): se borran primero las historias y después el proyecto. Las pruebas no dependen de
  HU-21 ni de HU-22: crean sus propios datos.

## Decision 14: Criterios de usabilidad de HU-04

- **Decision**: SC-009 y SC-010 se miden a mano en la Sprint Review con una prueba corta (ver
  `quickstart.md`) y se guardan en `evidencia-usabilidad.txt`.
- **Rationale**: Requieren personas; HU-04 los difirió hasta que existiera esta pantalla.
- **Pendiente**: confirmar los umbrales (90 % y 3 minutos) contra la spec de HU-04.