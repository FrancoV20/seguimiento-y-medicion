# Feature Specification: Cierre de Sprint

**Feature Branch**: `HU-15`

**Created**: 2026-10-21

**Status**: Implementada (pendiente de revisión en el Pull Request)

**Input**: User description: "HU-15 / Cierre de Sprint: Como Scrum Master, quiero cerrar el Sprint activo desde la web, para congelar sus datos y habilitar el cálculo de métricas. Depende de HU-14, HU-07, HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b."

> **Alcance**: HU-15 cierra el Sprint activo de un proyecto: lo pasa a "Finalizado", devuelve al Product
> Backlog las historias no completadas, conserva las completadas y guarda una foto de lo que el Sprint
> tenía. Lo hace desde una pantalla propia, "Cierre de Sprint", que muestra un resumen antes de cerrar.
> No incluye calcular la velocidad ni la desviación (HU-07), mostrar métricas (HU-25), consultar Sprints
> anteriores (HU-16) ni reabrir un Sprint.
>
> **Estados**: las historias tienen cuatro estados (decisión del equipo del 20/10/2026). Al cerrar, las
> "Hecho" quedan en el Sprint finalizado; las "En Sprint" y "En progreso" vuelven a "Pendiente" sin
> Sprint asignado (voto del equipo del 21/10/2026).
>
> **Foto del Sprint**: al devolver historias al Backlog, el Sprint cerrado perdería la información de lo
> que planificó. Por eso el cierre guarda, para todas sus historias, el estado final y los Story Points
> en la tabla `sprint_story_results`. HU-10, HU-16 y HU-25 la leen. Es el único cambio de esquema de
> esta historia (migración nueva).
>
> **Un solo Sprint activo**: la regla es de HU-04 (FIX-HU-04). Cerrar el Sprint es lo que permite crear
> el siguiente.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cerrar el Sprint (Priority: P1)

Como Scrum Master, quiero cerrar el Sprint activo y que las historias no completadas vuelvan al Backlog,
para congelar sus datos y planificar el siguiente Sprint.

**Why this priority**: Sin cerrar el Sprint no hay métricas definitivas (HU-07, HU-25) ni se puede crear
el siguiente Sprint (un solo Sprint activo).

**Independent Test**: Con un Sprint activo con historias "Hecho", "En progreso" y "En Sprint", cerrarlo;
verificar el Sprint, el estado y la asignación de cada historia, el registro del Sprint, que las
devueltas se pueden asignar a un Sprint nuevo y que HU-07 calcula la velocidad.

**Acceptance Scenarios**:

1. **Given** un Sprint activo con historias completadas y no completadas, **When** presiono "Cerrar
   Sprint", **Then** veo "Sprint N cerrado correctamente. Historias devueltas al Backlog: Y", el Sprint
   pasa a "Finalizado" y las historias no completadas vuelven al Product Backlog en estado
   "Pendiente", sin Sprint asignado, para poder asignarse a otro Sprint.
2. **Given** un Sprint activo con historias en "Hecho", **When** cierro el Sprint, **Then** las
   historias "Hecho" siguen en el Sprint finalizado con su estado y sus Story Points.
3. **Given** un Sprint activo con historias completadas y no completadas, **When** cierro el Sprint,
   **Then** queda registrado qué historias tenía, su estado final y sus Story Points, y ese registro no
   cambia aunque las historias devueltas se asignen a otro Sprint.
4. **Given** un Sprint activo sin ninguna historia en "Hecho", **When** presiono "Cerrar Sprint",
   **Then** el Sprint se cierra, todas sus historias vuelven al Backlog y su velocidad calculada por
   HU-07 es cero.
5. **Given** un Sprint cerrado con historias devueltas al Backlog, **When** creo un nuevo Sprint,
   **Then** las historias devueltas aparecen disponibles para asignar.

---

### User Story 2 - Rechazos y errores (Priority: P1)

Como Scrum Master, quiero que el cierre se bloquee si hay datos inválidos, si el Sprint ya está cerrado
o si algo falla, sin dejar cambios a medias, para no perder la coherencia del Sprint.

**Why this priority**: Un cierre parcial o con datos inválidos falsea la velocidad y las métricas, y no
se puede deshacer.

**Independent Test**: Cerrar un Sprint con datos inválidos, uno ya cerrado, con una falla de guardado y
con una carga de horas posterior; verificar mensajes y que nada cambia.

**Acceptance Scenarios**:

1. **Given** un Sprint con historias terminadas con horas reales negativas o Story Points inválidos,
   **When** presiono "Cerrar Sprint", **Then** el sistema aplica la validación de HU-07, bloquea el
   cierre, muestra qué historias deben corregirse y el Sprint sigue activo sin cambios.
2. **Given** un Sprint finalizado, **When** se intenta cerrarlo de nuevo, **Then** se rechaza con "El
   Sprint ya está cerrado" y no cambia nada.
3. **Given** un Sprint finalizado, **When** alguien intenta registrar horas en una de sus historias,
   **Then** el sistema lo rechaza (regla de HU-03).
4. **Given** un cierre válido, **When** ocurre un error al guardar, **Then** veo "La operación no pudo
   completarse", el Sprint sigue activo y ninguna historia ni registro cambia (sin cambios parciales).

---

### User Story 3 - Ver el resumen antes de cerrar (Priority: P1)

Como Scrum Master, quiero ver qué historias están completadas y cuáles volverán al Backlog antes de
cerrar, para decidir con la información a la vista, porque el cierre no se puede deshacer.

**Why this priority**: Es el punto de partida del cierre y evita cerrar por error.

**Independent Test**: Con un Sprint activo con historias en distintos estados, abrir la pantalla y
verificar el resumen; sin Sprint activo, verificar el mensaje.

**Acceptance Scenarios**:

1. **Given** un Sprint activo con historias completadas y no completadas, **When** abro "Cierre de
   Sprint", **Then** veo el Sprint, las historias completadas, las que volverán al Backlog, los Story
   Points completados sobre los planificados y el botón "Cerrar Sprint".
2. **Given** un proyecto sin Sprint activo, **When** abro "Cierre de Sprint", **Then** veo "No hay un
   Sprint activo para cerrar" y ningún botón de cierre.

---

### Edge Cases

- Un Sprint sin historias se puede cerrar: no devuelve ninguna y su registro queda vacío.
- Se puede cerrar antes de la fecha de fin; la fecha no condiciona el cierre.
- El formulario envía el identificador del Sprint que se veía. Si ese Sprint ya está finalizado, se
  rechaza con "El Sprint ya está cerrado" y no se cierra otro Sprint; si no existe o es de otro
  proyecto, "El Sprint no existe". El proyecto sale siempre del contexto del servidor.
- Si dos personas presionan "Cerrar Sprint" a la vez, solo una lo cierra; la otra ve "El Sprint ya está
  cerrado".
- El cierre bloquea las historias del Sprint mientras se hace, de modo que un cambio de estado de HU-14
  que llegue en ese momento se rechaza porque el Sprint ya está finalizado.
- Las historias devueltas conservan sus horas registradas y su estimación; cómo se cuentan las horas en
  el Sprint siguiente lo define HU-07.
- Una historia devuelta vuelve a ser elegible (`Pendiente` y sin asignación) y sus horas estimadas se
  pueden editar de nuevo (HU-17).
- Las historias "Hecho" mantienen su asignación al Sprint finalizado y por eso no se pueden asignar a
  otro Sprint ni editar sus horas estimadas.
- El registro del Sprint guarda los Story Points que la historia tenía al cerrar; una edición posterior
  no lo cambia.
- Si el Sprint tiene datos inválidos según HU-07, el cierre se bloquea completo; no hay cierre parcial.
- Si falla cualquier paso del cierre, se revierte todo: el Sprint sigue activo y las historias no
  cambian.
- Un Goal o un título con HTML o scripts se muestra como texto (regla de HU-20a y HU-20b).
- Recargar la página después de cerrar no cierra nada más ni repite el mensaje (patrón de HU-20b; T042).
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto y no se cierra nada (T043).
- Si falla la lectura del Sprint, se muestra la página de error de HU-20a (T044).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Cierre de Sprint" en el menú, que requiere un proyecto
  activo.
- **FR-002**: La pantalla MUST mostrar el Sprint activo, las historias completadas, las que volverán al
  Backlog y los Story Points completados sobre los planificados, con el botón "Cerrar Sprint".
- **FR-003**: Sin Sprint activo, la pantalla MUST mostrar "No hay un Sprint activo para cerrar" y ningún
  botón de cierre.
- **FR-004**: Cerrar MUST pasar el Sprint a "Finalizado" y mostrar "Sprint N cerrado correctamente.
  Historias devueltas al Backlog: Y" una sola vez.
- **FR-005**: Las historias que no estén en "Hecho" MUST volver a "Pendiente" y perder su asignación al
  Sprint, quedando elegibles para otro Sprint.
- **FR-006**: Las historias "Hecho" MUST conservar su estado, su asignación y sus Story Points en el
  Sprint finalizado.
- **FR-007**: El cierre MUST guardar, para cada historia que tenía el Sprint, su estado final y sus Story
  Points en `sprint_story_results`, y ese registro MUST NOT cambiar después.
- **FR-008**: Antes de cerrar, el sistema MUST invocar la validación de HU-07; si hay datos inválidos
  MUST bloquear el cierre, mostrar qué historias corregir y dejar el Sprint sin cambios.
- **FR-009**: Un Sprint sin historias completadas MUST poder cerrarse; su velocidad calculada por HU-07
  es cero.
- **FR-010**: Intentar cerrar un Sprint finalizado MUST rechazarse con "El Sprint ya está cerrado", sin
  cambios.
- **FR-011**: Después de cerrar, registrar horas en una historia del Sprint finalizado MUST rechazarse
  (regla de HU-03).
- **FR-012**: El cierre MUST ser todo o nada: Sprint, historias devueltas y registro del Sprint se
  guardan juntos o no se guarda nada; ante una falla MUST mostrarse "La operación no pudo completarse"
  sin éxito.
- **FR-013**: El proyecto MUST salir del contexto del servidor; el Sprint MUST venir del formulario
  pero se valida que pertenezca al proyecto activo y, si no existe, "El Sprint no existe".
- **FR-014**: Si falla la lectura del Sprint o de las historias, el sistema MUST mostrar la página de
  error de la aplicación y MUST NOT mostrar una pantalla incompleta.
- **FR-015**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.
- **FR-016**: Después de cerrar, el proyecto MUST poder crear un Sprint nuevo (regla de HU-04) y las
  historias devueltas MUST aparecer disponibles para asignar.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: cerrar el Sprint activo de un proyecto de forma atómica, devolviendo al Backlog las
historias no completadas, conservando las completadas y congelando lo que el Sprint tenía, para que las
métricas se calculen sobre datos que no cambian.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /cierre` | `sprint_id` | identificador numérico | sí | Sprint que se veía; debe ser del proyecto activo |

**Salidas esperadas**:

- Cierre válido: redirección 303 a `/cierre` y mensaje de una sola vista con el número del Sprint y la
  cantidad de historias devueltas.
- Cierre bloqueado por HU-07, Sprint ya cerrado: respuesta 409 con la pantalla y el mensaje (con la lista
  de historias a corregir en el primer caso).
- Identificador inválido o de otro proyecto: respuesta 422 con "El Sprint no existe".
- Falla de guardado: respuesta 500 con "La operación no pudo completarse" y el Sprint intacto.
- Falla de lectura al armar el resumen (`GET /cierre`): página de servicio no disponible de HU-20a (503).
- Cualquier falla durante el cierre (`POST /cierre`), incluida la lectura de las historias dentro de la
  transacción: respuesta 500 con "La operación no pudo completarse" y el Sprint intacto.
- Vista: Sprint, historias completadas, historias que volverán, Story Points y botón.

**Reglas de negocio**:

- Solo se cierra un Sprint activo del proyecto activo.
- La validación de HU-07 se aplica dentro de la transacción de cierre, después de bloquear el Sprint y sus
  historias, sobre las mismas historias que se cierran (las de `user_stories.active_sprint_id`).
- Las historias en "Hecho" quedan en el Sprint; las demás pasan a "Pendiente" y pierden la asignación.
- Se guarda el estado final y los Story Points de todas las historias del Sprint.
- El cierre es todo o nada.
- Un Sprint cerrado no se puede cerrar de nuevo ni recibir horas.

**Restricciones**:

- Una migración nueva (tabla `sprint_story_results`); sin dependencias nuevas. No se modifican archivos
  de HU-04, HU-14, HU-20a ni HU-20b; las modificaciones de HU-03 se limitan a la regla de carga
  posterior si no existiera (T001 y T016).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Mensaje | Respuesta |
|---|---|---|
| Datos inválidos según HU-07 | El de HU-07 con las historias a corregir | 409 |
| Sprint ya cerrado | "El Sprint ya está cerrado" | 409 |
| Sprint inexistente, no numérico o de otro proyecto | "El Sprint no existe" | 422 |
| Falla de guardado o de lectura durante el cierre (`POST`) | "La operación no pudo completarse" | 500 |
| Falla de lectura del resumen (`GET`) | Página de servicio no disponible de HU-20a | 503 |
| Sin proyecto activo válido | Mensaje de HU-20b | 303 |

Los textos propios de HU-15 son una propuesta del equipo y se definen en un solo lugar (`close_form.go`).

### Key Entities *(include if feature involves data)*

- **Sprint activo / finalizado**: Iteración del proyecto; el cierre la pasa a "Finalizado".
- **Historia devuelta**: Historia no completada que vuelve a "Pendiente" sin Sprint.
- **Resultado del Sprint** (`sprint_story_results`): Por cada historia del Sprint, su estado final y sus
  Story Points al cerrar.
- **Resumen de cierre**: Vista previa con completadas, devueltas y Story Points.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los cierres válidos deja el Sprint "Finalizado", las historias no completadas
  en "Pendiente" sin asignación y las completadas en el Sprint.
- **SC-002**: El 100% de los cierres registra el estado final y los Story Points de todas las historias
  del Sprint, y ese registro no cambia después.
- **SC-003**: El 100% de los cierres con datos inválidos, de Sprints ya cerrados o con fallas se
  rechaza sin cambios parciales.
- **SC-004**: Después de un cierre, el 100% de las historias devueltas puede asignarse a un Sprint
  nuevo.
- **SC-005**: Dos cierres simultáneos del mismo Sprint: solo uno se aplica.
- **SC-006**: El 100% de las cargas de horas posteriores en un Sprint finalizado se rechaza.
- **SC-007**: La pantalla responde en menos de 500 ms bajo condiciones normales (la prueba de T031 lo
  mide con la base local y un Sprint con menos de 50 historias).

## Assumptions

- HU-14 y FIX-HU-04 están integradas en `main`: las historias avanzan por los cuatro estados y hay un
  solo Sprint activo por proyecto.
- HU-07 expone la validación previa al cierre y el cálculo de velocidad; HU-15 solo los invoca (se
  verifica en T001).
- HU-03 rechaza cargar horas en historias que no están en el Sprint activo; si no lo hace, se agrega
  ahí (T001 y T016).
- El resumen reutiliza `BoardCatalog` de HU-14 para leer el Sprint activo y sus historias.
- HU-10, HU-16 y HU-25 leerán `sprint_story_results` para los Sprints finalizados; ese cambio es de cada
  una.
- No hay roles ni identificación para cerrar el Sprint; la autenticación y los roles se resuelven
  después del Sprint 2 (decisión del equipo).
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; el proyecto activo, de HU-20b.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Resumen antes de cerrar | FR-002 | T008, T031 | T036 | `close_handlers.go`, `close.html` |
| 2. Sin Sprint activo | FR-003 | T032 | T036 | `close.html` |
| 3. Cierre exitoso | FR-004, FR-005 | T006, T007, T012, T013, T017 | T017 | `src/sprint` (dominio), `close_handlers.go` |
| 4. Historias completadas conservadas | FR-006 | T006, T007, T013, T017 | T017 | `postgres_sprint_closer.go` |
| 5. Datos del Sprint congelados | FR-007 | T007, T015, T017 | T017 | `postgres_sprint_closer.go`, migración |
| 6. Cierre bloqueado por HU-07 | FR-008 | T006, T022, T027 | T027 | `src/sprint`, `close_form.go` |
| 7. Sprint sin historias completadas | FR-009 | T006, T014, T017 | T017 | `src/sprint` |
| 8. Sprint ya finalizado | FR-010, FR-013 | T006, T023, T024, T043 | T027 | `src/sprint`, `close_form.go`, HU-20b |
| 9. Historias devueltas disponibles | FR-016 | T015, T017 | T017 | `src/sprint` |
| 10. Carga de esfuerzo posterior | FR-011 | T016, T026, T027 | T027 | `src/effort` (HU-03) |
| 11. Error al cerrar | FR-012, FR-015 | T007, T025, T027, T030 | T027 | `postgres_sprint_closer.go`, `close_handlers.go` |
| (HU-20b) Recarga sin repetir | FR-004 | T042 | | `close_handlers.go` |
| (HU-20b) Sin proyecto activo | FR-013 | T043 | | `close_handlers.go` |
| (HU-20a) Error al leer | FR-014 | T033 | | `close_handlers.go` |