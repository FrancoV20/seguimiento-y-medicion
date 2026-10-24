# Feature Specification: Actualización de estado de historias (Tablero)

**Feature Branch**: `HU-14`

**Created**: 2026-10-21

**Status**: Approved

**Input**: User description: "HU-14 / Actualización de Estado y Registro de Historias Completadas: Como Product Builder, quiero cambiar el estado de una historia de mi Sprint activo (En Sprint → En progreso → Hecho) desde el tablero web, para registrar cuáles fueron completadas. Depende de HU-02, HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b."

> **Alcance**: HU-14 define la regla de las transiciones de estado de una historia del Sprint activo,
> la calcula en el dominio y la presenta en una pantalla propia, "Tablero", que muestra las historias
> del Sprint activo agrupadas por estado, con los Story Points completados sobre los planificados y
> un botón por acción. No incluye asignar ni quitar historias (HU-23), cerrar el Sprint (HU-15),
> calcular la velocidad (HU-07, que lee el estado "Hecho"), registrar horas (HU-03 y HU-24), ni
> identificar a la persona que hace el cambio.
>
> **Estados**: las historias tienen cuatro estados (Pendiente, En Sprint, En progreso, Hecho; decisión
> del equipo del 20/10/2026). Esta historia maneja las transiciones entre los tres últimos.
>
> **Un solo Sprint activo**: la regla es de HU-04 (FIX-HU-04). El tablero siempre trabaja sobre el
> Sprint activo del proyecto activo.
>
> **Punto que estaba "a definir"**: completar una historia sin horas registradas se permite con una
> advertencia. Es decisión del equipo.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cambiar el estado de una historia (Priority: P1)

Como Product Builder, quiero comenzar, completar y reabrir una historia de mi Sprint activo, para
reflejar el avance real y registrar cuáles fueron completadas.

**Why this priority**: Es el flujo principal: HU-15 (cierre), HU-07 (velocidad) y HU-25 (métricas)
necesitan historias en "Hecho" para tener datos.

**Independent Test**: Con un Sprint activo con historias "En Sprint", comenzar una, completarla con y
sin horas registradas y reabrirla; verificar el estado, los mensajes y los Story Points completados.

**Acceptance Scenarios**:

1. **Given** una historia "En Sprint" del Sprint activo, **When** presiono "Comenzar", **Then** veo
   "La historia pasó a En progreso" y la historia queda en estado "En progreso".
2. **Given** una historia "En progreso" del Sprint activo con horas registradas, **When** presiono
   "Completar", **Then** veo "La historia pasó a Hecho", la historia queda "Hecho" y sus Story Points
   se suman a los completados del Sprint.
3. **Given** una historia "En progreso" del Sprint activo sin horas registradas, **When** presiono
   "Completar", **Then** la historia pasa a "Hecho" y veo "La historia pasó a Hecho. Advertencia: no
   tiene horas reales registradas" en un solo mensaje.
4. **Given** una historia "Hecho" del Sprint activo, **When** presiono "Reabrir", **Then** veo "La
   historia volvió a En progreso", la historia pasa a "En progreso" y sus Story Points dejan de
   contarse como completados.

---

### User Story 2 - Rechazos y errores (Priority: P1)

Como Product Builder, quiero que los cambios inválidos o en conflicto se rechacen con un motivo y sin
tocar el estado, para no perder la coherencia de los datos del Sprint.

**Why this priority**: Un estado inconsistente o unos Story Points duplicados falsean la velocidad y
las métricas del Sprint.

**Independent Test**: Cambiar el estado de historias sin Sprint, de un Sprint finalizado, ya en
"Hecho", en "En Sprint" hacia "Hecho" y con el estado cambiado por otra persona; simular una falla de
guardado.

**Acceptance Scenarios**:

1. **Given** una historia sin Sprint o de un Sprint finalizado, **When** se intenta cambiar su
   estado, **Then** se rechaza con "La historia no está asignada a un Sprint activo" y el estado no
   cambia.
2. **Given** una historia que ya está en "Hecho", **When** se envía otra vez la acción de completarla,
   **Then** veo "La historia ya está en Hecho", el estado no cambia y los Story Points completados no
   se duplican.
3. **Given** una historia "En Sprint", **When** se intenta pasarla directamente a "Hecho", **Then** se
   rechaza con "No se puede pasar de En Sprint a Hecho" y el estado no cambia.
4. **Given** que otra persona cambió el estado de la historia mientras yo tenía el tablero abierto,
   **When** presiono una acción, **Then** no se cambia nada y veo "El estado de la historia cambió;
   revise el estado actual y vuelva a intentarlo".
5. **Given** una transición válida, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", el estado no cambia y no veo mensaje de éxito.

---

### User Story 3 - Ver el tablero del Sprint activo (Priority: P1)

Como Product Builder, quiero ver las historias del Sprint activo agrupadas por estado y con las
acciones que puedo hacer, para saber qué sigue y cuánto avanzó el Sprint.

**Why this priority**: Sin la pantalla no hay forma de cambiar el estado desde la web, y es el punto
de partida de cada acción.

**Independent Test**: Con un Sprint activo con historias en los tres estados, verificar los grupos,
los Story Points y las acciones; sin Sprint activo y con un Sprint sin historias, verificar los
mensajes.

**Acceptance Scenarios**:

1. **Given** un Sprint activo con historias en distintos estados, **When** abro "Tablero", **Then**
   veo el identificador del Sprint, las historias agrupadas por estado (En Sprint, En progreso, Hecho)
   y los Story Points completados sobre los planificados.
2. **Given** historias en distintos estados, **When** abro "Tablero", **Then** las historias "En
   Sprint" ofrecen solo "Comenzar", las "En progreso" solo "Completar" y las "Hecho" solo "Reabrir".
3. **Given** un proyecto sin Sprint activo, **When** abro "Tablero", **Then** veo "No hay un Sprint
   activo" y ninguna acción.
4. **Given** un Sprint activo sin historias asignadas, **When** abro "Tablero", **Then** veo el Sprint
   y "El Sprint no tiene historias asignadas".

---

### Edge Cases

- Una historia sin Story Points cuenta 0 en los completados y en los planificados, y se muestra "Sin
  estimar"; completarla es válido.
- Los Story Points planificados son la suma de todas las historias del Sprint y los completados, la de
  las historias en "Hecho" (misma definición que HU-10); no se guardan, siempre se calculan.
- Reabrir y volver a completar no duplica Story Points: el cálculo depende solo del estado actual.
- Las acciones son `comenzar`, `completar` y `reabrir`; cualquier otra se rechaza con "La acción no
  es válida".
- Un identificador de historia no numérico, inexistente o de otro proyecto se rechaza con "La
  historia no existe"; el proyecto sale siempre del contexto del servidor.
- Una historia "Pendiente" no pertenece a un Sprint, por lo que se rechaza como "no asignada a un
  Sprint activo".
- Una historia de un Sprint finalizado se rechaza aunque esté en "Hecho"; HU-15 decide qué pasa con
  las no terminadas al cerrar.
- Dos personas que presionan a la vez: el cambio se hace con una comparación atómica sobre el estado,
  y solo una gana; la otra ve "El estado de la historia cambió…".
- El estado que la persona veía viaja en el formulario (`estado_actual`); si falta, se omite esa
  comparación pero el cambio sigue siendo atómico.
- La advertencia de horas se calcula solo al completar, no al comenzar ni al reabrir.
- Cambiar el estado no modifica el Sprint, la asignación ni las horas.
- Si por datos anteriores a FIX-HU-04 hubiera dos Sprints activos, el tablero muestra el de mayor
  número.
- Un Goal o un título con HTML o scripts se muestra como texto (regla de HU-20a y HU-20b).
- Recargar la página después de un cambio no lo repite ni repite el mensaje (patrón de HU-20b; T044).
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto y no se cambia nada (T045).
- Si falla la lectura del Sprint o de las historias, se muestra la página de error de HU-20a (T046).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Tablero" en el menú, que requiere un proyecto activo.
- **FR-002**: El tablero MUST mostrar el Sprint activo del proyecto con sus historias agrupadas por
  estado (En Sprint, En progreso, Hecho) y los Story Points completados sobre los planificados.
- **FR-003**: Cada historia MUST ofrecer solo las acciones válidas para su estado ("En Sprint":
  Comenzar; "En progreso": Completar; "Hecho": Reabrir), tomadas de la regla del dominio.
- **FR-004**: Comenzar MUST pasar una historia de "En Sprint" a "En progreso" y mostrar "La historia
  pasó a En progreso".
- **FR-005**: Completar MUST pasar una historia de "En progreso" a "Hecho", mostrar "La historia pasó a
  Hecho" y sumar sus Story Points a los completados.
- **FR-006**: Completar una historia sin horas registradas MUST permitirse y MUST mostrar la
  advertencia en el mismo mensaje de éxito.
- **FR-007**: Reabrir MUST pasar una historia de "Hecho" a "En progreso", mostrar "La historia volvió a
  En progreso" y dejar de contar sus Story Points como completados.
- **FR-008**: El sistema MUST rechazar el cambio de una historia que no esté asignada a un Sprint
  activo con "La historia no está asignada a un Sprint activo", sin cambiar su estado.
- **FR-009**: Pedir el estado en el que la historia ya está MUST informar "La historia ya está en
  {estado}", sin cambiar nada ni duplicar Story Points.
- **FR-010**: Una transición que no sea una de las tres permitidas MUST rechazarse con "No se puede
  pasar de {origen} a {destino}", sin cambios.
- **FR-011**: Si el estado real difiere del que la persona veía, el sistema MUST rechazar el cambio con
  "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo", sin cambios; el
  cambio MUST hacerse de forma atómica respecto del estado esperado.
- **FR-012**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo completarse",
  MUST NOT mostrar éxito y MUST NOT cambiar el estado.
- **FR-013**: Sin Sprint activo, el tablero MUST mostrar "No hay un Sprint activo" y ninguna acción; con
  un Sprint sin historias, MUST mostrar "El Sprint no tiene historias asignadas".
- **FR-014**: El proyecto MUST salir del contexto del servidor y MUST NOT tomarse del formulario; una
  historia de otro proyecto o inexistente MUST rechazarse con "La historia no existe".
- **FR-015**: Una acción distinta de comenzar, completar o reabrir MUST rechazarse con "La acción no es
  válida".
- **FR-016**: Si falla la lectura del Sprint o de las historias, el sistema MUST mostrar la página de
  error de la aplicación y MUST NOT mostrar un tablero incompleto.
- **FR-017**: Los Story Points completados y planificados MUST calcularse desde el estado actual y MUST
  NOT guardarse.
- **FR-018**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: permitir que el equipo avance las historias del Sprint activo por sus estados desde una
pantalla web, con una única regla de transiciones en el dominio, sin duplicar ni perder Story Points
completados y sin cambios parciales ante errores.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /tablero/estado` | `historia_id` | identificador numérico | sí | Del proyecto activo |
| `POST /tablero/estado` | `accion` | texto | sí | `comenzar`, `completar` o `reabrir` |
| `POST /tablero/estado` | `estado_actual` | texto | no | Estado que la persona veía; si difiere del real se rechaza |

**Salidas esperadas**:

- Cambio válido: redirección 303 a `/tablero`, mensaje de una sola vista (éxito, o éxito con
  advertencia en el mismo mensaje).
- Estado ya aplicado: redirección 303 con mensaje de advertencia y sin cambios.
- Rechazo (no asignada a un Sprint activo, transición no permitida o estado desactualizado): respuesta
  409 con el tablero y el mensaje.
- Historia o acción inválida: respuesta 422 con el tablero y el mensaje.
- Falla de guardado: respuesta 500 con el tablero y "La operación no pudo completarse".
- Falla de lectura: página de error de HU-20a.
- Vista: Sprint, grupos por estado, Story Points `completados de planificados` y acciones.

**Reglas de negocio**:

- Transiciones permitidas: En Sprint → En progreso, En progreso → Hecho, Hecho → En progreso.
- Solo se cambia el estado de historias asignadas al Sprint activo del proyecto.
- Story Points completados = suma de los de las historias "Hecho"; planificados = suma de todas las del
  Sprint; ambos se calculan, no se guardan.
- Completar sin horas registradas se permite con advertencia.
- Orden de validación: historia del proyecto, Sprint activo, estado ya aplicado, estado esperado,
  transición permitida, estado real igual al origen de la acción.

**Restricciones**:

- Sin migraciones ni dependencias nuevas; no se modifican archivos de HU-04, HU-20a ni HU-20b.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Mensaje | Respuesta |
|---|---|---|
| Historia inexistente, no numérica o de otro proyecto | "La historia no existe" | 422 |
| Acción desconocida | "La acción no es válida" | 422 |
| Historia sin Sprint o de Sprint finalizado | "La historia no está asignada a un Sprint activo" | 409 |
| Estado ya aplicado | "La historia ya está en {estado}" | 303 |
| Transición no permitida | "No se puede pasar de {origen} a {destino}" | 409 |
| Estado desactualizado | "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo" | 409 |
| Falla de guardado | "La operación no pudo completarse" | 500 |
| Falla de lectura | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | Mensaje de HU-20b | 303 |

Los textos de los escenarios del Issue se respetan tal cual. Los de las respuestas 409, 422 y 500 y el de acción no válida los definió el equipo, no vienen del Issue, y se definen en un solo lugar (board_form.go).

### Key Entities *(include if feature involves data)*

- **Historia del Sprint activo**: Historia asignada al Sprint activo, con estado, Story Points y
  horas registradas.
- **Transición de estado**: Par origen → destino permitido por la regla del dominio.
- **Tablero**: Vista del Sprint activo con las historias agrupadas por estado y sus acciones.
- **Progreso del Sprint**: Story Points completados sobre planificados, calculado.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las transiciones permitidas sobre historias del Sprint activo actualiza el
  estado y muestra su mensaje.
- **SC-002**: El 100% de las transiciones no permitidas, sobre historias fuera de un Sprint activo o
  con estado desactualizado, se rechaza sin cambiar el estado.
- **SC-003**: Los Story Points completados siempre son iguales a la suma de los de las historias
  "Hecho", incluso tras reabrir o enviar dos veces la misma acción.
- **SC-004**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin éxito y
  sin cambiar el estado.
- **SC-005**: El 100% de las historias completadas sin horas registradas muestra la advertencia.
- **SC-006**: Ante dos cambios simultáneos sobre la misma historia, solo uno se aplica.
- **SC-007**: El tablero responde en menos de 500 ms bajo condiciones normales (la prueba de T032 lo
  mide con la base local y un Sprint con menos de 50 historias).

## Assumptions

- HU-04 y FIX-HU-04 están integradas en `main`: las historias asignadas quedan "En Sprint" y hay un
  solo Sprint activo por proyecto; si no, T001 lo detecta.
- La migración `004` ya admite los cuatro estados en `user_stories.status`; no hace falta migración.
- HU-07 define una historia completada como la que está en estado "Hecho"; HU-14 no lo cambia (se
  verifica en T001).
- Las horas registradas son las de HU-03 (`effort_logs`); esta historia solo comprueba si existen.
- Cerrar el Sprint y qué pasa con las historias no terminadas es de HU-15.
- No hay roles ni identificación para este cambio; la autenticación y los roles se resuelven después del
  Sprint 2 (decisión del equipo).
- Quitar una historia "En Sprint" es de HU-23; una historia "En progreso" no vuelve a "En Sprint".
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; el proyecto activo, de HU-20b.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Comenzar una historia | FR-004 | T005, T006, T014 | T018 | `src/sprint` (dominio), `board_handlers.go` |
| 2. Completar una historia | FR-005, FR-017 | T006, T015 | T018 | `src/sprint`, `board_handlers.go` |
| 3. Completar sin horas | FR-006 | T006, T016 | T018 | `src/sprint`, `board_form.go` |
| 4. Reabrir una historia | FR-007, FR-017 | T006, T017 | T018 | `src/sprint`, `board_handlers.go` |
| 5. Fuera de un Sprint activo | FR-008, FR-014 | T006, T023 | T028 | `src/sprint`, `board_form.go` |
| 6. Marcar dos veces | FR-009, FR-017 | T006, T024 | T028 | `src/sprint`, `board_form.go` |
| 7. Transición no permitida | FR-010, FR-015 | T005, T025 | T028 | `src/sprint`, `board_form.go` |
| 8. Estado desactualizado | FR-011 | T007, T026, T026b, T026c | T028 | `postgres_story_status_store.go`, `board_handlers.go` |
| 9. Error al guardar | FR-012, FR-018 | T027, T031 | T028 | `board_handlers.go` |
| 10. Tablero del Sprint activo | FR-002 | T009, T032 | T037 | `postgres_board_catalog.go`, `board.html` |
| 11. Acciones según el estado | FR-003 | T005, T033 | T037 | `src/sprint`, `board.html` |
| 12. Sin Sprint activo | FR-013 | T034 | T037 | `board.html` |
| 13. Sprint sin historias | FR-013 | T035 | T037 | `board.html` |
| (HU-20b) Recarga sin repetir | FR-001 | T044 | | `board_handlers.go` |
| (HU-20b) Sin proyecto activo | FR-014 | T045 | | `board_handlers.go` |
| (HU-20a) Error al leer | FR-016 | T036 | | `board_handlers.go` |