# Feature Specification: Interfaz web de registro de esfuerzo

**Feature Branch**: `HU-24`

**Created**: 2026-10-22

**Status**: Draft

**Input**: User description: "HU-24 / Interfaz Web de Registro de Esfuerzo (presenta HU-03): Como Product Builder, quiero registrar mis horas desde un formulario web, para cargar el esfuerzo sin comandos. Depende de HU-03, HU-04, FIX-HU-04, HU-20a y HU-20b."

> **Alcance**: HU-24 presenta HU-03 en el navegador: una pantalla "Registro de esfuerzo" con las historias del
> Sprint activo, el formulario de carga y el historial de la historia elegida. Las reglas de las horas, la
> actividad, la fecha y el integrante son de HU-03; esta historia las muestra junto a cada campo y no las copia.
> No incluye editar ni borrar registros, métricas (HU-07, HU-25) ni cambiar estados de historias (HU-14).
>
> **Integrante**: el registro queda a nombre del integrante identificado en HU-20b. El formulario no tiene
> campo de integrante. Por eso la sección exige proyecto activo e integrante identificado.
>
> **Sprint activo**: las historias ofrecidas son las asignadas al Sprint activo del proyecto (un solo Sprint
> activo por proyecto, FIX-HU-04). Las devueltas al Backlog por HU-15 y las de Sprints finalizados no aparecen.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar horas desde la web (Priority: P1)

Como Product Builder, quiero elegir una historia del Sprint activo y cargar fecha, actividad y horas, para
registrar mi esfuerzo sin comandos.

**Why this priority**: Sin esta pantalla el esfuerzo real solo se puede cargar desde el núcleo, y HU-07 no
puede calcular la desviación desde la web.

**Independent Test**: Con un proyecto con un Sprint activo, historias asignadas y un integrante identificado,
cargar horas y verificar el mensaje, el registro (integrante, fecha, actividad, horas) y el acumulado.

**Acceptance Scenarios**:

1. **Given** una historia del Sprint activo y un integrante identificado, **When** completo fecha (por defecto
   hoy), actividad y horas y guardo, **Then** veo "Esfuerzo registrado correctamente", el registro queda a
   nombre del integrante identificado y veo el acumulado de la historia actualizado.
2. **Given** historias en distintos estados de Sprint, **When** abro el selector, **Then** solo aparecen las del
   Sprint activo del proyecto.
3. **Given** un proyecto sin Sprint activo, **When** abro "Registro de esfuerzo", **Then** veo "No hay un Sprint
   activo" y no veo el formulario.
4. **Given** que no me identifiqué como integrante, **When** entro a "Registro de esfuerzo", **Then** se me pide
   identificarme y, al hacerlo, vuelvo a la pantalla y no se registra nada hasta entonces.

---

### User Story 2 - Errores y rechazos (Priority: P1)

Como Product Builder, quiero ver todos los errores de la carga juntos, cada uno junto a su campo y con los
mensajes de HU-03, para corregirlos de una sola vez sin perder lo que escribí.

**Why this priority**: Un formulario que muestra un error por vez o pierde los datos desalienta la carga diaria,
de la que dependen las métricas.

**Independent Test**: Enviar el formulario con varios campos inválidos, con una historia ajena al Sprint activo y
con una falla de guardado; verificar mensajes, datos conservados y que no se registra nada.

**Acceptance Scenarios**:

1. **Given** un formulario con varios campos inválidos, **When** guardo, **Then** veo todos los mensajes exactos
   de HU-03, cada uno junto a su campo, los datos se conservan y no se registra nada.
2. **Given** una historia que no pertenece al Sprint activo del proyecto, **When** se envía una carga sobre ella,
   **Then** veo el error junto al campo de historia, los datos se conservan y no se registra nada.
3. **Given** una carga válida, **When** falla el guardado, **Then** veo "La operación no pudo completarse", los
   datos se conservan y no se muestra éxito.

---

### User Story 3 - Ver el historial de una historia (Priority: P2)

Como Product Builder, quiero ver quién cargó qué horas en una historia y el total, para saber cuánto esfuerzo
lleva registrado.

**Why this priority**: Confirma que la carga quedó bien y muestra el acumulado que HU-07 usa; la carga funciona
sin ella.

**Independent Test**: Con una historia con registros de dos integrantes, abrirla y verificar la lista y el
total; con una historia sin registros, verificar el mensaje.

**Acceptance Scenarios**:

1. **Given** una historia con registros, **When** la abro, **Then** veo la lista de cargas (integrante, fecha,
   actividad, horas) con las más recientes primero y el total.
2. **Given** una historia del Sprint activo sin horas cargadas, **When** la abro, **Then** veo "Todavía no hay
   horas registradas en esta historia" y el total en cero.

---

### Edge Cases

- El integrante se toma siempre de la identificación del servidor; un campo de integrante enviado en el formulario
  se ignora.
- Si la identificación falta o es de un integrante ajeno al proyecto, HU-20b pide identificarse (no se pierde
  nada que no estuviera guardado; lo escrito en el formulario no se conserva en ese desvío).
- Un `historia` no numérico, inexistente, de otro proyecto o fuera del Sprint activo en la dirección se ignora: se
  muestra la pantalla sin historial.
- Si el Sprint se cierra entre que se abre el formulario y se envía, HU-03 rechaza la carga y el error se muestra
  junto al campo de historia.
- Una historia en "Hecho" del Sprint activo se ofrece y acepta cargas, igual que HU-03 (T001 lo confirma).
- La fecha por defecto es la del día del servidor; no se valida contra las fechas del Sprint ni contra el futuro
  porque HU-03 no lo exige.
- Dos envíos iguales registran dos cargas: no hay detección de duplicados (HU-03 no la define). Recargar la página
  después de guardar no reenvía nada (HU-20b).
- Las horas con coma o con punto se aceptan; un valor que no es número se informa con el mensaje de HU-03 de
  horas inválidas.
- Una actividad con HTML o scripts se muestra como texto (regla de HU-20b).
- Una historia sin horas muestra acumulado en cero, no "Sin estimar".
- Registrar horas no cambia el estado de la historia (eso es de HU-14).
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto y no se registra nada.
- Si falla la lectura de las historias o del historial, se muestra la página de error de HU-20a.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Registro de esfuerzo" en el menú, que requiere proyecto activo e
  integrante identificado.
- **FR-002**: La pantalla MUST mostrar las historias del Sprint activo del proyecto con sus horas reales
  acumuladas y el formulario de carga (historia, fecha, actividad y horas).
- **FR-003**: El campo de fecha MUST traer la fecha de hoy por defecto.
- **FR-004**: El selector y la lista MUST incluir solo historias del Sprint activo del proyecto activo.
- **FR-005**: Una carga válida MUST registrarse mediante HU-03, redirigir con 303 a la pantalla de esa historia,
  mostrar "Esfuerzo registrado correctamente" una sola vez y mostrar el acumulado actualizado.
- **FR-006**: Con datos inválidos el sistema MUST mostrar todos los errores juntos, cada uno junto a su campo,
  con los mensajes exactos de HU-03, conservar los datos y no registrar nada. Las reglas de horas, actividad,
  fecha e integrante MUST aplicarse en HU-03 y no copiarse en la capa web.
- **FR-007**: Una historia faltante o que no pertenece al Sprint activo del proyecto MUST rechazarse con un error
  junto al campo de historia y sin registrar nada.
- **FR-008**: El integrante MUST salir de la identificación del servidor y no del formulario; sin integrante
  identificado MUST aplicarse la regla de HU-20b y, si una carga llega sin integrante, HU-03 la rechaza.
- **FR-009**: La pantalla MUST mostrar el historial de la historia elegida con integrante, fecha, actividad y
  horas, las más recientes primero, y el total.
- **FR-010**: Una historia sin registros MUST mostrar "Todavía no hay horas registradas en esta historia" y total
  en cero.
- **FR-011**: Sin Sprint activo la pantalla MUST mostrar "No hay un Sprint activo" y MUST NOT mostrar el
  formulario.
- **FR-012**: Ante una falla de guardado el sistema MUST mostrar "La operación no pudo completarse", conservar los
  datos y no mostrar éxito.
- **FR-013**: Ante una falla de lectura el sistema MUST mostrar la página de error de la aplicación y MUST NOT
  mostrar una pantalla incompleta.
- **FR-014**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.
- **FR-015**: Los textos ingresados (actividad, títulos de historias) MUST mostrarse escapados como texto.
- **FR-016**: El proyecto MUST salir del contexto del servidor y todas las lecturas MUST filtrarse por él.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: presentar el registro de esfuerzo de HU-03 en una pantalla web que lista las historias del Sprint
activo, valida el formulario mostrando todos los errores juntos y muestra el historial y el acumulado de cada
historia.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /esfuerzo` | `historia` | identificador numérico | sí | Debe ser del Sprint activo del proyecto |
| `POST /esfuerzo` | `fecha` | `AAAA-MM-DD` | sí (regla de HU-03) | Por defecto hoy en el formulario |
| `POST /esfuerzo` | `actividad` | texto | sí (regla de HU-03) | |
| `POST /esfuerzo` | `horas` | número con coma o punto | sí (regla de HU-03) | |
| `GET /esfuerzo` | `historia` (opcional) | identificador numérico | no | Elige el historial a mostrar |

El integrante y el proyecto no son campos: salen del contexto del servidor.

**Salidas esperadas**:

- Carga válida: redirección 303 a `/esfuerzo?historia=ID`, mensaje de una sola vista y acumulado actualizado.
- Datos inválidos o historia no disponible: respuesta 422 con el formulario, los datos conservados y los errores
  junto a cada campo.
- Sin Sprint activo al enviar: respuesta 409 con el mensaje y sin formulario.
- Falla de guardado: respuesta 500 con "La operación no pudo completarse" y los datos conservados.
- Falla de lectura: página de error de HU-20a.
- Vista: historias del Sprint activo con horas acumuladas, formulario e historial con total.

**Reglas de negocio**:

- Solo se ofrecen y aceptan historias del Sprint activo del proyecto activo.
- Las reglas de horas, actividad, fecha e integrante son de HU-03 y se muestran con sus mensajes.
- El acumulado de una historia es la suma de las horas de todos sus registros.
- El historial muestra los registros de todos los integrantes, los más recientes primero.

**Restricciones**:

- Sin migraciones ni dependencias nuevas. Esta historia no escribe en `effort_logs`: llama al servicio de HU-03.
- No se modifican archivos de HU-04, HU-14, HU-15, HU-20a ni HU-20b; en HU-03 solo se agrega una función que
  devuelve todas las violaciones si no existiera (T001 y T010).
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Mensaje | Respuesta |
|---|---|---|
| Varios campos inválidos | Los de HU-03 junto a cada campo | 422 |
| Historia no seleccionada | "Seleccione una historia" | 422 |
| Historia fuera del Sprint activo | "La historia no pertenece al Sprint activo" | 422 |
| Fecha con formato inválido | "La fecha no es válida" | 422 |
| Sin Sprint activo al enviar | "No hay un Sprint activo" | 409 |
| Sin integrante identificado | Mensaje de HU-20b | 303 |
| Falla de guardado | "La operación no pudo completarse" | 500 |
| Falla de lectura | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | Mensaje de HU-20b | 303 |

Los textos propios de HU-24 son una propuesta del equipo y se definen en un solo lugar (`effort_form.go`).

### Key Entities *(include if feature involves data)*

- **Registro de esfuerzo**: Carga de horas de HU-03 (integrante, historia, fecha, actividad, horas).
- **Historia del Sprint activo**: Historia asignada al Sprint activo del proyecto, con sus horas acumuladas.
- **Historial de la historia**: Lista de registros de una historia con el total.
- **Formulario de carga**: Historia, fecha, actividad y horas tal como se enviaron.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las cargas válidas queda registrado a nombre del integrante identificado y se refleja en
  el acumulado.
- **SC-002**: El 100% de los envíos con datos inválidos muestra todos los errores juntos, conserva los datos y no
  registra nada.
- **SC-003**: Ninguna historia fuera del Sprint activo del proyecto aparece en el selector ni recibe cargas.
- **SC-004**: El total del historial es igual a la suma de sus registros y coincide con el acumulado de la lista.
- **SC-005**: Ninguna falla de guardado muestra un mensaje de éxito.
- **SC-006**: La pantalla responde en menos de 500 ms bajo condiciones normales (la prueba de T032 lo mide con la
  base local, un Sprint con menos de 50 historias y una historia con menos de 200 registros).

## Assumptions

- HU-03 expone un servicio de registro y las reglas de validación con mensajes propios; HU-24 los reutiliza (se
  verifica en T001). Si HU-03 devuelve solo el primer error, T010 agrega una función que devuelve todos.
- `effort_logs` (HU-03) tiene columnas para historia, integrante, fecha, actividad y horas; los nombres exactos se
  verifican en T001 y T004.
- HU-20b ya identifica al integrante y expone su identificador y nombre en el contexto, y pide identificarse al
  entrar a una sección que lo requiere (T002).
- HU-04 y FIX-HU-04 mantienen un solo Sprint activo por proyecto y sus asignaciones consistentes.
- HU-03 acepta cargas sobre cualquier historia asignada al Sprint activo, incluida una en "Hecho" (T001).
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los errores, de
  HU-20a; el proyecto y el integrante activos, de HU-20b.
- No hay contraseñas ni roles: la autenticación y los roles se resuelven después del Sprint 2 (decisión del
  equipo).

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Carga exitosa | FR-002, FR-003, FR-005, FR-008 | T013, T014 | T018 | `effort_handlers.go`, `effort.html` |
| 2. Errores juntos | FR-006 | T008, T009, T023, T024 | T028 | `effort_form.go`, `effort_actions.go` |
| 3. Historial de la historia | FR-009 | T006, T032 | T036 | `postgres_effort_catalog.go`, `effort.html` |
| 4. Historia sin registros | FR-010 | T033 | T036 | `effort.html` |
| 5. Historias disponibles | FR-004, FR-016 | T006, T015 | T018 | `postgres_effort_catalog.go` |
| 6. Sin Sprint activo | FR-011 | T016 | T018 | `effort_handlers.go`, `effort.html` |
| 7. Sin integrante identificado | FR-001, FR-008 | T017 | T018 | `src/cmd/server/main.go` |
| 8. Historia no disponible | FR-007 | T025, T027 | T028 | `effort_form.go`, `effort_handlers.go` |
| 9. Falla de almacenamiento | FR-012, FR-014 | T026, T031 | T028 | `effort_handlers.go` |
| (HU-20b) Recarga sin reenviar | FR-005 | T043 | | `effort_handlers.go` |
| (HU-20b) Sin proyecto activo | FR-016 | T044 | | `effort_handlers.go` |
| (HU-20b) HTML como texto | FR-015 | T035 | | `effort.html` |
| (HU-20a) Error al leer | FR-013 | T038, T045 | | `effort_handlers.go` |