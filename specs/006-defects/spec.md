# Feature Specification: Registro y trazabilidad de defectos

**Feature Branch**: `HU-06`

**Created**: 2026-10-26

**Status**: Implementada (pendiente de revisión en el Pull Request)

**Input**: User description: "HU-06 / Registro y Trazabilidad de Defectos: Como Product Builder, quiero registrar un defecto indicando su descripción, severidad e historia relacionada, para darle seguimiento desde su detección hasta su resolución. Depende de HU-02, HU-04, FIX-HU-04, HU-20a y HU-20b."

> **Alcance**: HU-06 permite registrar defectos y verlos en una pantalla "Defectos": formulario (descripción,
> severidad, historia relacionada) y lista de los defectos del proyecto. Un defecto nace "Abierto" y con el
> Sprint activo como "Sprint de detección". Resolver y reabrir es de HU-19; contar defectos por Sprint es de
> HU-10. No incluye editar ni borrar defectos.
>
> **Sin Sprint activo**: se bloquea el registro, igual que en HU-19 (decisión del equipo para HU-19; para HU-06
> se confirma). El Sprint de detección nunca se elige a mano.
>
> **Severidad**: Crítica, Alta, Media y Baja (propuesta del equipo; se cambia en un solo lugar).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar un defecto (Priority: P1)

Como Product Builder, quiero cargar un defecto con su descripción, su severidad y la historia que lo originó,
para dejar trazado cuándo y dónde apareció.

**Why this priority**: Sin defectos registrados no hay seguimiento de calidad ni métricas de defectos (HU-10,
HU-08, HU-09).

**Independent Test**: Con un Sprint activo y una historia, registrar un defecto y verificar el mensaje, el
estado "Abierto", el Sprint de detección y que la historia no cambia; repetir con una historia "Hecho" de un
Sprint anterior.

**Acceptance Scenarios**:

1. **Given** que encontré un bug probando una funcionalidad, **When** cargo el defecto ingresando su descripción
   y su "Severidad" y selecciono la historia de usuario que lo originó, **Then** veo "Defecto registrado
   correctamente. Sprint de detección: Sprint N" y el defecto queda guardado con estado "Abierto", asociado a
   esa historia y al Sprint activo como "Sprint de detección".
2. **Given** una historia "Hecho" de un Sprint finalizado del proyecto, **When** registro un defecto sobre ella,
   **Then** el defecto queda "Abierto" con el Sprint activo como Sprint de detección y la historia no cambia de
   estado.

---

### User Story 2 - Errores y rechazos (Priority: P1)

Como Product Builder, quiero ver todos los errores juntos y que no se guarde nada inválido, para corregir el
formulario sin perder lo que escribí.

**Why this priority**: Un defecto sin severidad, sin historia o sin Sprint distorsiona las métricas de calidad.

**Independent Test**: Enviar el formulario con campos faltantes, con una historia ajena, sin Sprint activo y con
una falla de guardado; verificar mensajes, datos conservados y que no se guarda nada.

**Acceptance Scenarios**:

1. **Given** un Product Builder reportando una anomalía, **When** pretende confirmar el almacenamiento omitiendo
   la Severidad o la historia asociada, **Then** la aplicación bloquea la acción, exige ingresar los elementos
   requeridos con un mensaje junto a cada campo y conserva lo ingresado.
2. **Given** un Product Builder reportando una anomalía, **When** deja la descripción vacía o con solo espacios o
   supera los 1000 caracteres, **Then** la aplicación bloquea la acción, indica el problema junto al campo y no
   se guarda nada.
3. **Given** un proyecto sin Sprint activo, **When** abro "Defectos" o intento registrar un defecto, **Then**
   veo "No hay un Sprint activo; no se puede registrar un defecto", no veo el formulario y no se guarda nada.
4. **Given** una historia que no pertenece al proyecto activo o que no existe, **When** se envía el defecto con
   esa historia, **Then** veo el error junto al campo de historia, los datos se conservan y no se guarda nada.
5. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo completarse",
   los datos se conservan y el defecto no aparece.

---

### User Story 3 - Ver los defectos del proyecto (Priority: P2)

Como Product Builder, quiero ver la lista de defectos del proyecto, para saber cuáles hay, de qué historia son y
cuándo se detectaron.

**Why this priority**: Confirma que el registro quedó bien y es el punto de partida de HU-19; el registro
funciona sin ella.

**Independent Test**: Con defectos de dos proyectos, abrir la pantalla de cada uno y verificar columnas, orden y
aislamiento; sin defectos, verificar el mensaje.

**Acceptance Scenarios**:

1. **Given** un proyecto con defectos, **When** abro "Defectos", **Then** veo cada uno con identificador,
   descripción, severidad, estado, historia relacionada, Sprint de detección y Sprint de resolución, con los más
   recientes primero.
2. **Given** un proyecto sin defectos, **When** abro "Defectos", **Then** veo "Aún no hay defectos" y el
   formulario para registrar el primero.

---

### Edge Cases

- Un defecto sin Sprint de resolución muestra "—" en esa columna; HU-19 lo completa al resolver.
- La descripción se guarda sin espacios en los bordes; una descripción de solo espacios es vacía y una de 1000
  caracteres exactos es válida, una de 1001 no.
- Una descripción con el carácter NUL (`%00`) o con bytes que no son UTF-8 válido (por ejemplo `%FF`) se rechaza
  con 422 y "La descripción contiene caracteres no válidos": es un dato del usuario, no un error del servidor
  (PostgreSQL no admite esos bytes en TEXT).
- Dos defectos con la misma descripción se aceptan: no hay detección de duplicados.
- El Sprint de detección es el Sprint activo del proyecto al confirmar; un campo de Sprint enviado en el
  formulario se ignora.
- Si el Sprint se cierra mientras se confirma el defecto, el registro y el cierre (HU-15) no se mezclan: o el
  defecto queda en ese Sprint antes del cierre, o se rechaza porque ya no hay Sprint activo.
- La historia relacionada puede estar en cualquier estado y en cualquier Sprint del proyecto (incluso uno
  finalizado o ninguno); registrar un defecto no cambia su estado ni su asignación.
- Una historia devuelta al Backlog por HU-15 sigue siendo válida como historia relacionada.
- Un `historia` no numérico, inexistente o de otro proyecto se rechaza con un error junto a ese campo.
- Una severidad fuera de la lista (enviada a mano) se rechaza con "La severidad no es válida".
- Una descripción con HTML o scripts se muestra como texto (regla de HU-20b).
- No se registra quién reportó el defecto; no hay identificación de integrante.
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto y no se guarda nada.
- Si falla la lectura de la lista o de las historias, se muestra la página de error de HU-20a.
- Recargar la página después de registrar no crea un segundo defecto ni repite el mensaje (HU-20b).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Defectos" en el menú, que requiere un proyecto activo.
- **FR-002**: La pantalla MUST ofrecer un formulario con descripción, severidad (Crítica, Alta, Media o Baja) e
  historia relacionada.
- **FR-003**: Un registro válido MUST guardar el defecto con estado "Abierto", la historia relacionada, el Sprint
  activo del proyecto como Sprint de detección y sin Sprint de resolución, y MUST mostrar "Defecto registrado
  correctamente. Sprint de detección: Sprint N" una sola vez.
- **FR-004**: La historia relacionada MUST ser una historia del proyecto activo, de cualquier estado; registrar
  el defecto MUST NOT cambiar su estado ni su asignación.
- **FR-005**: La descripción, la severidad y la historia MUST ser obligatorias; ante datos faltantes o inválidos
  el sistema MUST mostrar todos los errores juntos, cada uno junto a su campo, conservar los datos y no guardar
  nada.
- **FR-006**: La descripción MUST guardarse sin espacios en los bordes, tener entre 1 y 1000 caracteres y ser
  texto UTF-8 válido sin el carácter NUL.
- **FR-007**: Sin Sprint activo el sistema MUST bloquear el registro, mostrar "No hay un Sprint activo; no se
  puede registrar un defecto" y MUST NOT mostrar el formulario.
- **FR-008**: Una historia inexistente, no numérica o de otro proyecto MUST rechazarse con un error junto al campo
  de historia y sin guardar nada.
- **FR-009**: La pantalla MUST listar los defectos del proyecto con identificador, descripción, severidad,
  estado, historia relacionada, Sprint de detección y Sprint de resolución, los más recientes primero.
- **FR-010**: Sin defectos la pantalla MUST mostrar "Aún no hay defectos".
- **FR-011**: Ante una falla de guardado el sistema MUST mostrar "La operación no pudo completarse", conservar los
  datos, no mostrar éxito y no guardar nada.
- **FR-012**: Ante una falla de lectura el sistema MUST mostrar la página de error de la aplicación y MUST NOT
  mostrar una pantalla incompleta.
- **FR-013**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.
- **FR-014**: Los textos ingresados MUST mostrarse escapados como texto.
- **FR-015**: El proyecto y el Sprint de detección MUST salir del servidor y no del formulario.
- **FR-016**: El registro MUST ser atómico respecto del cierre de Sprint: el Sprint de detección es el que está
  activo al confirmar.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: registrar defectos asociados a una historia y al Sprint activo, con su severidad y estado inicial
"Abierto", y mostrarlos en una pantalla web, para dar trazabilidad desde la detección hasta la resolución.

**Entradas**:

| Formulario | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `POST /defectos` | `descripcion` | texto | sí | 1 a 1000 caracteres sin espacios en los bordes; UTF-8 válido, sin NUL |
| `POST /defectos` | `severidad` | `Crítica`, `Alta`, `Media`, `Baja` | sí | |
| `POST /defectos` | `historia` | identificador numérico | sí | Historia del proyecto activo |

El proyecto y el Sprint de detección no son campos: salen del servidor.

**Salidas esperadas**:

- Registro válido: redirección 303 a `/defectos` y mensaje de una sola vista con el número del Sprint.
- Datos inválidos o historia ajena: respuesta 422 con el formulario, los datos conservados y los errores junto a
  cada campo.
- Sin Sprint activo: respuesta 409 con el mensaje y sin formulario.
- Falla de guardado: respuesta 500 con "La operación no pudo completarse" y los datos conservados.
- Falla de lectura: página de error de HU-20a.
- Vista: formulario (si hay Sprint activo) y lista de defectos.

**Reglas de negocio**:

- El defecto nace "Abierto", sin Sprint de resolución, con el Sprint activo como Sprint de detección.
- La severidad es una de las cuatro de la lista.
- La historia pertenece al proyecto activo; su estado y su asignación no cambian.
- Sin Sprint activo no se registra.
- El registro y el cierre de Sprint no se mezclan.

**Restricciones**:

- Una migración nueva (tabla `defects`); sin dependencias nuevas. No se modifican archivos de HU-02, HU-04,
  HU-15, HU-20a ni HU-20b.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Mensaje | Respuesta |
|---|---|---|
| Descripción vacía o solo espacios | "La descripción es obligatoria" | 422 |
| Descripción de más de 1000 caracteres | "La descripción no puede superar 1000 caracteres" | 422 |
| Descripción con carácter NUL o UTF-8 inválido | "La descripción contiene caracteres no válidos" | 422 |
| Severidad vacía | "Seleccione la severidad" | 422 |
| Severidad fuera de la lista | "La severidad no es válida" | 422 |
| Historia vacía | "Seleccione la historia relacionada" | 422 |
| Historia inexistente, no numérica o de otro proyecto | "La historia no pertenece al proyecto" | 422 |
| Sin Sprint activo | "No hay un Sprint activo; no se puede registrar un defecto" | 409 |
| Falla de guardado | "La operación no pudo completarse" | 500 |
| Falla de lectura | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | Mensaje de HU-20b | 303 |

Los textos propios de HU-06 son una propuesta del equipo y se definen en un solo lugar (`defects_form.go`).

### Key Entities *(include if feature involves data)*

- **Defecto**: Anomalía registrada con descripción, severidad, estado ("Abierto" o "Resuelto"), historia
  relacionada, Sprint de detección y Sprint de resolución (vacío hasta HU-19).
- **Severidad**: Crítica, Alta, Media o Baja.
- **Historia relacionada**: Historia del proyecto que originó el defecto.
- **Sprint de detección**: Sprint activo al momento de registrar el defecto.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los registros válidos queda con estado "Abierto", el Sprint activo como Sprint de
  detección y sin Sprint de resolución.
- **SC-002**: El 100% de los envíos inválidos muestra todos los errores juntos, conserva los datos y no guarda
  nada.
- **SC-003**: Ningún defecto se registra sin Sprint activo ni con una historia de otro proyecto.
- **SC-004**: Un registro y un cierre de Sprint simultáneos nunca dejan un defecto con un Sprint inconsistente.
- **SC-005**: La lista muestra los defectos del proyecto activo y ninguno de otro proyecto.
- **SC-006**: La pantalla responde en menos de 500 ms bajo condiciones normales (la prueba de T032 lo mide con la
  base local y menos de 200 defectos).

## Assumptions

- HU-04 y FIX-HU-04 mantienen un solo Sprint activo por proyecto, con estado `'Activo'`; HU-15 lo pasa a
  `'Finalizado'`.
- Las historias están en `user_stories` con `project_id`, `title` y `status` (HU-02).
- HU-10 contará los defectos por Sprint de detección y de resolución; HU-19 completará la resolución; ese cambio es
  de cada una.
- Los niveles de severidad (Crítica, Alta, Media y Baja) y el límite de 1000 caracteres son propuesta del equipo.
- No hay roles ni identificación de quien reporta; la autenticación y los roles se resuelven después del Sprint 2
  (decisión del equipo).
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los errores, de
  HU-20a; el proyecto activo, de HU-20b.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Criterios | Tests | BDD | Código |
|---|---|---|---|---|---|
| Sección Defectos protegida por proyecto | FR-001 | | T023, T043 | | `src/cmd/server/main.go`, `src/web/defects_handlers.go` |
| 1. Vinculación de defecto a historia | FR-002, FR-003, FR-015 | SC-001 | T006, T007, T016, T018 | T019 | `src/defect`, `src/web/defects_handlers.go`, `src/web/templates/defects.html` |
| 2. Omisión de datos obligatorios | FR-005 | SC-002 | T006, T014, T024 | T029 | `src/web/defects_form.go`, `src/defect/service.go` |
| 3. Descripción obligatoria y límite | FR-005, FR-006 | SC-002 | T006, T025, T052 | T029, T052 | `src/defect/service.go`, `src/web/defects_form.go` |
| 4. Sin Sprint activo y cierre atómico | FR-007, FR-016 | SC-003, SC-004 | T007, T027 | T029 | `src/defect/postgres_store.go`, `src/web/defects_handlers.go`, `src/web/templates/defects.html` |
| 5. Historia de otro proyecto | FR-008 | SC-003 | T007, T026 | T029 | `src/defect/postgres_store.go`, `src/web/defects_form.go` |
| 6. Historia de un Sprint anterior | FR-004 | SC-001 | T017 | T019 | `src/defect/postgres_store.go` |
| 7. Lista de defectos | FR-009, FR-014 | SC-005, SC-006 | T012, T032, T034 | T035 | `src/web/postgres_defects_catalog.go`, `src/web/templates/defects.html` |
| 8. Lista vacía | FR-010 | | T033 | T035 | `src/web/templates/defects.html` |
| 9. Error al guardar | FR-011, FR-013 | SC-002 | T007, T028, T031 | T029 | `src/web/defects_handlers.go`, `src/web/defects_form.go` |
| (HU-20b) Recarga sin duplicar | FR-003 | SC-001 | T042 | | `src/web/defects_handlers.go` |
| (HU-20b) Sin proyecto activo | FR-015 | | T043 | | `src/web/defects_handlers.go` |
| (HU-20a) Error al leer | FR-012 | | T037, T044 | | `src/web/defects_handlers.go` |