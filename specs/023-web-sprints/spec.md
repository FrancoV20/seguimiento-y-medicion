# Feature Specification: Interfaz web de Sprints

**Feature Branch**: `HU-23`

**Created**: 2026-10-20 | **Revised**: 2026-10-21

**Status**: Draft

**Input**: User description: "HU-23 / Interfaz Web de Sprints (presenta HU-04): Como Scrum Master, quiero crear un Sprint y agregar o quitar sus historias desde la web, para planificar la iteración visualmente. Depende de HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b."

> **Alcance**: HU-23 presenta en el navegador lo que HU-04 ya resuelve (creación de Sprints con
> identificador automático, Sprint Goal y período, y asignación exclusiva de historias). En una sola
> pantalla, "Sprints", muestra el Sprint activo del proyecto, permite crear uno cuando no hay y
> agregarle o quitarle historias. No incluye cerrar ni editar Sprints (HU-15), consultar Sprints
> anteriores (HU-16), cambiar el estado de las historias (HU-14) ni mover historias entre Sprints.
>
> **Un solo Sprint activo**: un proyecto trabaja con un Sprint activo a la vez. La regla es de HU-04
> y se implementa en **FIX-HU-04** (decisión del equipo del 20/10/2026); la pantalla no la aplica:
> no ofrece el formulario cuando hay un activo y, si igual llega un envío, muestra el rechazo del
> dominio. HU-23 no se puede integrar antes de ese fix.
>
> **Estados**: las historias tienen cuatro estados (Pendiente, En Sprint, En progreso, Hecho). Asignar
> deja la historia en "En Sprint" y quitarla la devuelve a "Pendiente" (FIX-HU-04).
>
> **Quitar historias**: forma parte de esta historia. `RemoveStory` de HU-04 no valida estado ni
> horas, así que HU-23 agrega esa validación en el dominio (T047).
>
> **Pendientes de HU-04 que se cierran acá**: los criterios de usabilidad (crear un Sprint y asignar
> historias al primer intento y en menos de 3 minutos; SC-009 y SC-010).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crear un Sprint con historias (Priority: P1)

Como Scrum Master, quiero completar el Sprint Goal y las fechas, marcar historias del Backlog y
crear el Sprint, para planificar la iteración sin comandos.

**Why this priority**: Es el flujo principal de HU-04 hecho consumible: sin Sprint activo con
historias, HU-14 (estados), HU-24 (esfuerzo) y HU-25 (métricas) no tienen con qué trabajar.

**Independent Test**: Con un proyecto con historias pendientes y sin Sprint activo, crear un Sprint
con Goal, fechas y tres historias; verificar el identificador, las historias "En Sprint" y el
mensaje de éxito; repetir sin historias y con un Sprint activo existente.

**Acceptance Scenarios**:

1. **Given** un proyecto con historias pendientes y sin Sprint activo, **When** completo Sprint Goal y
   fechas válidas, marco historias de la lista y presiono "Crear", **Then** veo "Sprint creado
   correctamente" y el Sprint aparece como activo con un identificador automático ("Sprint N") que no
   puedo editar y con sus historias asignadas en estado "En Sprint".
2. **Given** un proyecto sin Sprint activo, **When** completo Sprint Goal y fechas válidas, no marco
   ninguna historia y presiono "Crear", **Then** el Sprint se crea y veo "Sprint creado
   correctamente. Todavía no tiene historias asignadas".
3. **Given** que el proyecto tiene el Sprint activo "Sprint N", **When** abro "Sprints", **Then** veo
   el Sprint activo, no veo el formulario de creación y veo "Ya hay un Sprint activo (Sprint N)".

---

### User Story 2 - Elegir historias y corregir errores (Priority: P1)

Como Scrum Master, quiero ver qué historias puedo asignar y por qué otras no, y que cada error
aparezca junto a su campo sin perder lo que escribí, para corregir sin volver a cargar todo.

**Why this priority**: Un Sprint con datos inválidos o con historias que no estaban disponibles no
sirve para planificar, y la exclusividad de las historias es una regla dura del equipo.

**Independent Test**: Abrir la lista con historias asignadas y no pendientes; enviar sin Goal, con
fechas inválidas, faltantes o mal formadas; asignar una historia que dejó de estar disponible;
simular una falla de guardado y un envío sin proyecto activo.

**Acceptance Scenarios**:

1. **Given** una historia asignada a un Sprint o que no está pendiente, **When** abro la lista para
   asignar historias, **Then** aparece deshabilitada con el motivo ("Ya está asignada al Sprint N" o
   "No está pendiente").
2. **Given** que marqué una historia que otra persona asignó mientras armaba el formulario, **When**
   presiono "Crear", **Then** no se crea el Sprint, veo el mensaje de HU-04 que indica cuál historia no
   está disponible y los datos y las demás marcas se conservan.
3. **Given** que dejo el Sprint Goal vacío o con solo espacios y el resto de los datos son válidos,
   **When** presiono "Crear", **Then** veo el mensaje de HU-04 que pide el Sprint Goal junto al campo,
   los datos y las historias marcadas se conservan y no se crea el Sprint.
4. **Given** que ingreso una fecha de fin anterior o igual a la de inicio y el resto de los datos son
   válidos, **When** presiono "Crear", **Then** veo el mensaje de fechas de HU-04 junto a la fecha de
   fin, los datos y las historias marcadas se conservan y no se crea el Sprint.
5. **Given** que dejo vacía la fecha de inicio o la de fin y el resto de los datos son válidos,
   **When** presiono "Crear", **Then** veo "Falta la fecha de inicio" o "Falta la fecha de fin", según
   cuál falte, junto a ese campo, los datos y las historias marcadas se conservan y no se crea el
   Sprint.
6. **Given** un envío del formulario con una fecha que no tiene el formato AAAA-MM-DD, **When** se
   procesa, **Then** veo "La fecha no tiene un formato válido" junto a esa fecha, los datos y las
   historias marcadas se conservan y no se crea el Sprint.
7. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", los datos y las marcas se conservan, no veo mensaje de éxito y no queda ningún Sprint
   ni asignación a medias.

---

### User Story 3 - Ver el Sprint activo (Priority: P1)

Como Scrum Master, quiero ver el Sprint activo con su objetivo, fechas e historias, para saber en
qué trabaja el equipo.

**Why this priority**: Confirma el resultado de la planificación y es el punto de partida para
agregar o quitar historias.

**Independent Test**: Con un Sprint activo con historias en distintos estados, abrir la pantalla y
verificar identificador, Goal, fechas e historias; sin Sprint activo, verificar el mensaje y el
formulario; simular una falla de lectura.

**Acceptance Scenarios**:

1. **Given** un Sprint activo con historias en distintos estados, **When** abro "Sprints", **Then** veo
   su identificador, su Sprint Goal, sus fechas y las historias asignadas con su estado.
2. **Given** un proyecto sin Sprint activo, **When** abro "Sprints", **Then** veo "No hay un Sprint
   activo" y el formulario para crear el primero.

---

### User Story 4 - Agregar historias al Sprint activo (Priority: P2)

Como Scrum Master, quiero agregar historias a un Sprint ya creado, para completar la planificación
cuando lo creé sin historias o me faltó alguna.

**Why this priority**: Sin esto, un Sprint creado vacío (que HU-04 permite) no se puede completar
desde la web.

**Independent Test**: Con un Sprint activo y historias pendientes, marcar algunas y agregarlas;
repetir sin marcar nada y con una historia que dejó de estar disponible.

**Acceptance Scenarios**:

1. **Given** un Sprint activo e historias disponibles, **When** marco historias y presiono "Agregar
   historias", **Then** veo "Historias agregadas correctamente" y las historias aparecen en el Sprint
   con estado "En Sprint".
2. **Given** un Sprint activo, **When** presiono "Agregar historias" sin marcar ninguna, **Then** veo
   "Elija al menos una historia" y no cambia nada.

---

### User Story 5 - Quitar historias del Sprint activo (Priority: P2)

Como Scrum Master, quiero quitar del Sprint una historia que todavía no se empezó, para corregir un
error de planificación sin recrear el Sprint.

**Why this priority**: Un error de planificación no se podría corregir desde la web, y quitar una
historia empezada dejaría horas huérfanas.

**Independent Test**: Quitar una historia "En Sprint" sin horas; verificar que vuelve a "Pendiente";
intentar quitar una historia en progreso, hecha o con horas registradas, desde la pantalla y con un
envío directo.

**Acceptance Scenarios**:

1. **Given** una historia en estado "En Sprint" sin horas registradas, **When** presiono "Quitar",
   **Then** veo "Historia quitada del Sprint", la historia deja de estar en el Sprint, vuelve a
   "Pendiente" y puedo asignarla de nuevo.
2. **Given** una historia en progreso, hecha o con horas registradas, **When** veo el Sprint,
   **Then** "Quitar" aparece deshabilitado con el motivo ("Ya está en progreso", "Ya está hecha" o
   "Tiene horas registradas").
3. **Given** una historia en progreso, hecha o con horas registradas, **When** se envía igualmente
   la acción de quitarla, **Then** se rechaza con el motivo y la historia sigue en el Sprint sin
   cambios.

---

### Edge Cases

- Una historia está disponible solo si está en estado "Pendiente" y no tiene asignación a ningún
  Sprint; el motivo mostrado para las demás es "Ya está asignada al Sprint N" o "No está pendiente".
- Un identificador de Sprint enviado en el formulario se ignora: el identificador lo genera el
  sistema y el Sprint sobre el que se trabaja sale del proyecto activo.
- Un identificador de historia repetido en el envío se toma una sola vez; uno no numérico o de otro
  proyecto se rechaza con un mensaje junto a la lista.
- Crear un Sprint con la fecha de fin igual a la de inicio se rechaza (regla de HU-04); una fecha de
  inicio pasada se acepta, porque HU-04 no lo prohíbe.
- Un Sprint cuyo período se superpone con otro no se valida: solo puede haber un Sprint activo.
- Si con el formulario abierto otra persona crea el Sprint activo, el envío se rechaza por la regla
  de FIX-HU-04 con "Ya hay un Sprint activo (Sprint N)", sin crear nada ni consumir un número.
- Si por datos anteriores a FIX-HU-04 un proyecto tuviera dos Sprints activos, la pantalla muestra el
  de mayor número y no ofrece crear otro.
- Los Sprints finalizados y sus historias no se muestran en esta pantalla (HU-16).
- Quitar la última historia deja el Sprint vacío y es válido.
- Una historia de otro proyecto o de otro Sprint no se puede quitar ni agregar: se rechaza.
- Si la selección de proyecto se pierde al guardar, los datos del formulario no se conservan; se pide
  elegir el proyecto de nuevo (limitación aceptada en HU-20b).
- Un Goal con HTML o scripts se muestra como texto (regla de HU-20a/HU-20b).
- Si ocurre un error en la asignación al crear el Sprint, HU-04 garantiza que no queden asociaciones
  ni Sprint parciales.
- Sin historias en el Backlog, la lista muestra "El Backlog no tiene historias" y el Sprint se puede
  crear vacío.
- Recargar la página después de crear un Sprint no crea un segundo Sprint ni repite el mensaje
  (patrón de HU-20b; se prueba en T013).
- Si falla la lectura del Sprint o de las historias, se muestra la página de error de la aplicación y
  no una pantalla incompleta (HU-20a; se prueba en T032).
- Sin proyecto activo válido al guardar, HU-20b redirige a la selección de proyecto con su mensaje y
  no se guarda nada (se prueba en T024).
- Si una historia marcada al agregar dejó de estar disponible, no se asigna ninguna (se prueba en
  T038).
- Entre la lectura y el quitar alguien podría cargar horas; la validación del dominio vuelve a
  comprobarlas al quitar, y la carrera restante es aceptada.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer el acceso a la sección "Sprints" desde el menú, que requiere
  un proyecto activo.
- **FR-002**: El formulario de creación MUST pedir Sprint Goal, fecha de inicio, fecha de fin y
  permitir marcar historias del Backlog del proyecto; el identificador MUST NOT ser editable.
- **FR-003**: Una creación válida MUST crear el Sprint según las reglas de HU-04 (identificador
  secuencial "Sprint N", estado activo), asignar las historias marcadas con estado "En Sprint",
  mostrar "Sprint creado correctamente" una sola vez y mostrar el Sprint.
- **FR-004**: Sin historias marcadas, el sistema MUST crear el Sprint y MUST mostrar "Sprint creado
  correctamente. Todavía no tiene historias asignadas".
- **FR-005**: Cuando el proyecto ya tenga un Sprint activo, el sistema MUST NOT ofrecer el formulario
  de creación y MUST informar "Ya hay un Sprint activo (Sprint N)"; ante un envío igualmente
  recibido, MUST mostrar ese rechazo del dominio sin crear nada.
- **FR-006**: La lista de historias MUST permitir marcar solo las disponibles y MUST mostrar el resto
  deshabilitado con el motivo.
- **FR-007**: Si una historia marcada deja de estar disponible, el sistema MUST rechazar la creación
  completa, informar cuáles no están disponibles y conservar los datos.
- **FR-008**: Ante un Goal vacío o una fecha de fin anterior o igual a la de inicio, el sistema MUST
  mostrar los mensajes de HU-04 junto a cada campo, conservar los datos y MUST NOT crear el Sprint.
- **FR-009**: Ante una fecha faltante o de formato inválido, el sistema MUST mostrar un mensaje junto
  al campo que lo indique y MUST NOT crear el Sprint.
- **FR-010**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo completarse",
  conservar los datos, MUST NOT mostrar éxito y MUST NOT dejar Sprint ni asignaciones parciales.
- **FR-011**: Cuando no haya un proyecto activo válido al guardar, el sistema MUST bloquear la acción
  y redirigir a la selección de proyecto con el mensaje de HU-20b.
- **FR-012**: El proyecto y el Sprint sobre los que se opera MUST salir del contexto del servidor y
  MUST NOT tomarse de campos del formulario.
- **FR-013**: La pantalla MUST mostrar el Sprint activo con su identificador, Sprint Goal, fechas y
  las historias asignadas con su estado.
- **FR-014**: Cuando no haya un Sprint activo, el sistema MUST mostrar "No hay un Sprint activo" y el
  formulario de creación.
- **FR-015**: Si falla la lectura del Sprint o de las historias, el sistema MUST mostrar la página de
  error de la aplicación y MUST NOT mostrar una pantalla incompleta.
- **FR-016**: Recargar la página después de crear un Sprint MUST NOT crear un segundo Sprint.
- **FR-017**: El sistema MUST permitir agregar al Sprint activo las historias disponibles marcadas,
  con estado "En Sprint", asignándolas todas o ninguna, y mostrar "Historias agregadas
  correctamente".
- **FR-018**: Al agregar sin marcar historias, el sistema MUST informar "Elija al menos una historia"
  y MUST NOT cambiar nada.
- **FR-019**: El sistema MUST permitir quitar del Sprint activo una historia en estado "En Sprint" y
  sin horas registradas, devolviéndola a "Pendiente" y mostrando "Historia quitada del Sprint".
- **FR-020**: Para una historia en progreso, hecha o con horas registradas, el sistema MUST mostrar
  "Quitar" deshabilitado con el motivo y MUST rechazar un envío igualmente recibido, con el motivo y
  sin cambios. La regla MUST aplicarse en el dominio y la pantalla MUST usar la misma función.
- **FR-021**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: permitir crear el Sprint activo de un proyecto, ver su estado y agregar o quitar
historias desde una pantalla web, con las reglas de HU-04 y FIX-HU-04 y sin que la persona pierda lo
que escribió ante un error.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /sprints` | `objetivo` | texto | sí | Vacío o solo espacios se trata como vacío |
| `POST /sprints` | `fecha_inicio`, `fecha_fin` | fecha `AAAA-MM-DD` | sí | Fin posterior al inicio |
| `POST /sprints` | `historia` | identificador numérico (uno por historia marcada) | no | Los repetidos se toman una vez |
| `POST /sprints/historias` | `historia` | idem | sí, al menos una | Se asignan todas o ninguna |
| `POST /sprints/historias/quitar` | `historia_id` | identificador numérico | sí | Debe estar en el Sprint activo del proyecto activo |

**Salidas esperadas**:

- Envío válido: redirección 303 a `/sprints`, mensaje de una sola vista y el Sprint actualizado.
- Datos inválidos: respuesta 422 con el formulario, los datos y las marcas conservados y un mensaje
  junto a cada campo con error.
- Conflicto (ya hay un Sprint activo, historia no disponible o no se puede quitar): respuesta 409 con
  el motivo y los datos conservados.
- Falla de guardado: respuesta 500 con "La operación no pudo completarse" y los datos conservados.
- Falla de lectura: página de error de HU-20a.
- Vista: identificador, Goal, fechas `dd/mm/aaaa` e historias con su estado y su botón "Quitar".

**Reglas de negocio**:

- Un proyecto tiene como máximo un Sprint activo (HU-04, FIX-HU-04); la pantalla no lo aplica, solo
  informa el rechazo del dominio.
- Una historia está disponible si está "Pendiente" y sin asignación. Asignar la deja "En Sprint" y
  quitarla la devuelve a "Pendiente".
- Se puede quitar solo una historia "En Sprint" y sin horas registradas (regla de dominio agregada en
  T047). Las demás se rechazan con el motivo.
- Crear y agregar asignan todas las historias o ninguna.

**Restricciones**:

- Sin migraciones ni dependencias nuevas; no se modifican HU-20a ni HU-20b. En HU-04 solo se agrega
  la validación de quitar (T047).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Campo | Mensaje | Respuesta |
|---|---|---|---|
| Goal vacío | `objetivo` | El de HU-04 | 422 |
| Fin anterior o igual al inicio | `fecha_fin` | El de HU-04 | 422 |
| Fecha vacía | la fecha afectada | "Falta la fecha de inicio" o "Falta la fecha de fin" | 422 |
| Formato de fecha inválido | la fecha afectada | "La fecha no tiene un formato válido" | 422 |
| Identificador de historia no numérico o de otro proyecto | `historia` | Mensaje propio junto a la lista | 422 |
| Ya hay un Sprint activo | general | "Ya hay un Sprint activo (Sprint N)" | 409 |
| Historia no disponible | `historia` | El de HU-04, con las historias rechazadas | 409 |
| Agregar sin marcar | `historia` | "Elija al menos una historia" | 422 |
| Quitar historia no removible | general | "Ya está en progreso", "Ya está hecha" o "Tiene horas registradas" | 409 |
| Falla de guardado | general | "La operación no pudo completarse" | 500 |
| Falla de lectura | página | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | redirección | Mensaje de HU-20b | 303 |

Los mensajes de HU-04 se toman del código (T001). Los textos propios de HU-23 son una propuesta del
equipo y se definen en un solo lugar (`sprints_form.go`).

### Key Entities *(include if feature involves data)*

- **Sprint activo**: Iteración en curso del proyecto, con identificador secuencial, Goal, fechas e
  historias; definida por HU-04.
- **Historia disponible**: Historia del Backlog en estado "Pendiente" y sin asignación a un Sprint.
- **Asignación**: Relación exclusiva entre una historia y un Sprint; deja la historia en "En Sprint".
- **Proyecto activo**: Proyecto elegido en HU-20b sobre el que trabaja la pantalla.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los Sprints creados con datos válidos aparece como Sprint activo con su
  identificador secuencial y sus historias en "En Sprint".
- **SC-002**: El 100% de las entradas inválidas se rechaza con el mensaje junto a su campo, conservando
  los datos, sin crear el Sprint y sin consumir un número de Sprint.
- **SC-003**: El 100% de los intentos de crear un Sprint con otro activo en el proyecto es rechazado
  con el número del activo.
- **SC-004**: El 100% de las historias no disponibles aparece deshabilitado con su motivo y no se
  asigna nunca.
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin éxito
  y sin Sprint ni asignaciones parciales.
- **SC-006**: El 100% de los intentos de quitar una historia en progreso, hecha o con horas es
  rechazado sin cambios, tanto desde la pantalla como con un envío directo.
- **SC-007**: El 100% de los guardados sin proyecto activo válido se bloquea y redirige a la selección
  de proyecto sin guardar nada.
- **SC-008**: La pantalla responde en menos de 500 ms bajo condiciones normales de uso (la prueba de
  T030 la mide con la base local y un Sprint con menos de 50 historias).
- **SC-009**: Al menos el 90% de las personas de prueba crea un Sprint y asigna historias al primer
  intento con datos válidos. *(Se verifica manualmente en la Sprint Review; confirmar el umbral contra
  la spec de HU-04.)*
- **SC-010**: Crear un Sprint y asignar tres historias toma menos de 3 minutos. *(Se verifica
  manualmente en la Sprint Review; confirmar el umbral contra la spec de HU-04.)*

## Assumptions

- El modelo, las validaciones, la secuencia de identificadores y la asignación exclusiva de HU-04 no
  cambian, salvo FIX-HU-04 y la validación de quitar que agrega esta historia; los mensajes de error
  son los de HU-04.
- FIX-HU-04 está integrada en `main` antes de integrar HU-23; si no lo estuviera, T012 falla.
- "En Sprint" es el estado que HU-04 asigna a una historia al incorporarla al Sprint. HU-14 debe
  partir de ese estado y no de "Pendiente".
- Cerrar un Sprint, devolver las historias no terminadas al Backlog y los Sprints anteriores son de
  HU-15 y HU-16; para que una historia no terminada pueda volver a asignarse, el cierre debe eliminar
  su asignación (la relación es única por historia).
- Mover historias entre Sprints no aplica con un solo Sprint activo; "Quitar" y "Agregar" cubren la
  corrección de la planificación.
- La edición del Goal y de las fechas de un Sprint ya creado queda fuera de alcance.
- Las horas registradas de una historia son las de HU-03; esta historia solo comprueba si existen.
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; la selección del proyecto activo, de HU-20b.
- No hay roles: cualquier persona con un proyecto activo puede planificar Sprints.
- Crear historias es de HU-22; esta pantalla las lista del Backlog pero no las crea.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Crear Sprint con historias | FR-001 a FR-003, FR-012 | T009, T010 | T014 | `sprints_form.go`, `sprints_handlers.go`, `sprints.html` |
| 2. Sprint sin historias | FR-004 | T011 | T014 | `sprints_handlers.go` |
| 3. Ya hay un Sprint activo | FR-005 | T012 | T014 | `sprints_form.go`, `sprints_handlers.go`, `sprints.html` |
| 4. Sin Sprint activo | FR-014 | T031 | T033 | `sprints.html` |
| 5. Vista del Sprint activo | FR-013 | T006, T030 | T033 | `postgres_sprint_catalog.go`, `sprints.html` |
| 6. Historia no disponible en la lista | FR-006 | T006, T019 | T025 | `postgres_sprint_catalog.go`, `sprints.html` |
| 7. Historia que dejó de estar disponible | FR-007 | T020 | T025 | `sprints_form.go` |
| 8. Sprint Goal vacío | FR-008 | T021 | T025 | `sprints_form.go` |
| 9. Fin anterior o igual al inicio | FR-008 | T021 | T025 | `sprints_form.go` |
| 10. Fecha faltante | FR-009 | T009, T022 | T025 | `sprints_form.go` |
| 11. Fecha con formato inválido | FR-009 | T009, T022 | T025 | `sprints_form.go` |
| 12. Error al guardar el Sprint | FR-010, FR-021 | T023, T029 | T025 | `sprints_handlers.go` |
| 13. Agregar historias | FR-017 | T036 | T039 | `sprints_handlers.go`, `sprints.html` |
| 14. Agregar sin marcar ninguna | FR-018 | T037 | T039 | `sprints_handlers.go` |
| 15. Quitar una historia | FR-019 | T043, T044 | T046 | `src/sprint` (dominio), `sprints_handlers.go`, `sprints.html` |
| 16. Historia que no se puede quitar | FR-020 | T043, T045 | T046 | `src/sprint` (dominio), `sprints.html` |
| 17. Intento de quitar una historia que no se puede | FR-020 | T043, T045 | T046 | `src/sprint` (dominio), `sprints_handlers.go` |
| (HU-20b) Recarga sin duplicar | FR-016 | T013 | | `sprints_handlers.go` |
| (HU-20b) Sin proyecto activo al guardar | FR-011, FR-012 | T024 | | `sprints_handlers.go` |
| (HU-20a) Error al leer | FR-015 | T032 | | `sprints_handlers.go` |