# Feature Specification: Interfaz web del Product Backlog

**Feature Branch**: `HU-22`

**Created**: 2026-10-18 | **Revised**: 2026-10-21

**Status**: Draft

**Input**: User description: "HU-22 / Interfaz Web del Product Backlog (presenta HU-02): Como Scrum Master, quiero cargar historias en el Backlog desde un formulario web, para planificar el trabajo sin comandos. Depende de HU-02, HU-20a y HU-20b."

> **Alcance**: HU-22 presenta en el navegador lo que HU-02 ya resuelve (modelo, validaciones, alta y
> persistencia de historias). Muestra el formulario de alta y una lista mínima de las historias del
> proyecto activo en una sola pantalla, "Backlog". El orden por prioridad, el filtro por estado y la
> edición de historias son de HU-13 y quedan fuera. Incluye el campo opcional de horas estimadas
> del alta; modificarlas después es de HU-17.
>
> **Contexto**: la pantalla trabaja siempre sobre el proyecto activo de HU-20b. Sin proyecto activo
> válido no se muestra ni se guarda nada: se redirige a la selección de proyecto.
>
> **Regla de horas estimadas**: HU-02 ya guarda las horas estimadas al crear la historia (columna
> `estimated_hours`, mayor que 0). Los límites de dos decimales y de 9999,99 (el máximo de la columna
> `NUMERIC(6,2)`) y la coma decimal se validan en el dominio, en el mismo lugar que HU-02, para que
> haya una sola regla que HU-17 reutilice al modificar las horas.
>
> **Estados de las historias**: una historia nueva siempre queda "Pendiente". Los estados "En Sprint",
> "En progreso" y "Hecho" los cambian HU-04, HU-23 y HU-14; esta pantalla solo los muestra.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cargar una historia desde el formulario web (Priority: P1)

Como Scrum Master, quiero completar título, descripción, prioridad, criterios de aceptación y, si los
tengo, Story Points y horas estimadas, y guardar la historia, para planificar el trabajo sin comandos.

**Why this priority**: Es el flujo principal de HU-02 hecho consumible: sin historias en el Backlog
no se pueden armar Sprints (HU-23) ni cargar esfuerzo (HU-24).

**Independent Test**: Con un proyecto activo, completar el formulario con datos válidos y verificar
que la historia aparece en el Backlog con estado "Pendiente"; agregar y quitar criterios, dejar
criterios en blanco, guardar sin Story Points y sin horas, e ingresar horas con coma.

**Acceptance Scenarios**:

1. **Given** un proyecto seleccionado, **When** completo título, descripción, prioridad, al menos un
   criterio de aceptación, Story Points y horas estimadas y guardo, **Then** veo "Historia creada
   correctamente" y la historia aparece en el Backlog con estado "Pendiente" y los valores
   ingresados.
2. **Given** el formulario de historia con datos escritos, **When** presiono "Agregar criterio",
   **Then** el formulario se vuelve a mostrar con un campo de criterio más, conserva lo ya escrito y
   no se guarda nada.
3. **Given** el formulario de historia con más de un campo de criterio, **When** presiono "Quitar" en
   uno de ellos, **Then** ese campo desaparece, el resto de lo escrito se conserva y no se guarda
   nada.
4. **Given** el formulario de historia con un solo campo de criterio, **When** presiono "Quitar" en
   ese campo, **Then** el campo sigue en el formulario, lo escrito se conserva y no se guarda nada.
5. **Given** que dejo campos de criterio en blanco y al menos uno tiene texto, **When** guardo,
   **Then** los campos en blanco se ignoran y la historia se guarda solo con los criterios que tienen
   texto.
6. **Given** el formulario de historia con los demás datos válidos, **When** guardo sin Story Points,
   **Then** la historia se crea, veo "Historia creada correctamente. Falta completar la estimación de
   Story Points" y en el Backlog figura con Story Points "Sin estimar".
7. **Given** el formulario de historia con los demás datos válidos, **When** guardo sin horas
   estimadas, **Then** la historia se crea, veo "Historia creada correctamente" sin advertencia y en
   el Backlog figura con horas estimadas "Sin estimar".
8. **Given** el formulario de historia con los demás datos válidos, **When** ingreso "1,5" como horas
   estimadas y guardo, **Then** la historia se crea con 1,5 horas estimadas y el Backlog muestra 1,5.

---

### User Story 2 - Corregir errores sin perder los datos (Priority: P1)

Como Scrum Master, quiero ver cada error junto a su campo y conservar lo que escribí, para
corregirlo sin volver a cargar todo.

**Why this priority**: Una historia sin criterios de aceptación o con datos inválidos no sirve para
planificar, y un formulario que borra los datos al fallar desalienta la carga.

**Independent Test**: Enviar el formulario sin criterios, con cada campo inválido y con todos juntos;
simular una falla de guardado; enviar sin proyecto activo. Verificar mensajes, datos conservados y
que no se crea nada.

**Acceptance Scenarios**:

1. **Given** el formulario de historia con los demás datos válidos, **When** guardo sin ningún
   criterio con texto, **Then** veo el mensaje de HU-02 que pide al menos un criterio de aceptación
   junto al campo de criterios, los datos se conservan y no se crea la historia.
2. **Given** que dejo el título vacío o con solo espacios y el resto de los datos son válidos,
   **When** guardo, **Then** veo el mensaje de HU-02 que indica que el título es obligatorio junto al
   título, los datos se conservan y no se crea la historia.
3. **Given** que dejo la descripción vacía o con solo espacios y el resto de los datos son válidos,
   **When** guardo, **Then** veo el mensaje de HU-02 que indica que la descripción es obligatoria
   junto a la descripción, los datos se conservan y no se crea la historia.
4. **Given** un envío del formulario con una prioridad distinta de Alta, Media o Baja, **When** se
   procesa, **Then** veo el mensaje de HU-02 que indica que la prioridad no es válida junto a la
   prioridad, los datos se conservan y no se crea la historia.
5. **Given** un envío del formulario con Story Points fuera de 1, 2, 3, 5, 8, 13, 21, 34, 55 y 89,
   **When** se procesa, **Then** veo un mensaje junto a los Story Points que indica que no son un
   valor válido, los datos se conservan y no se crea la historia.
6. **Given** que ingreso horas estimadas negativas, iguales a cero, no numéricas o con más de dos
   decimales, **When** guardo, **Then** veo "El valor de horas no es válido" junto al campo, los datos se conservan y no se crea la historia.
7. **Given** el formulario de historia con los demás datos válidos, **When** ingreso horas estimadas
   mayores que 9999,99 y guardo, **Then** veo "Las horas estimadas no pueden superar 9999,99" junto al
   campo, los datos se conservan y no se crea la historia.
8. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", los datos se conservan, no veo mensaje de éxito y la historia no aparece en el
   Backlog.

---

### User Story 3 - Ver el Backlog del proyecto activo (Priority: P2)

Como Scrum Master, quiero ver las historias del proyecto en la misma pantalla, para confirmar lo que
cargué y saber qué hay antes de agregar otra.

**Why this priority**: Permite verificar el resultado del alta y orienta a quien planifica, pero no
bloquea la carga.

**Independent Test**: Con dos proyectos con historias distintas, abrir el Backlog de cada uno y
verificar columnas y aislamiento; con un proyecto sin historias, verificar el mensaje y el
formulario; simular una falla de lectura.

**Acceptance Scenarios**:

1. **Given** un proyecto seleccionado con historias, **When** abro "Backlog", **Then** veo cada
   historia con su identificador, título, prioridad, Story Points, horas estimadas y estado, y "Sin
   estimar" donde falte un valor, no cero.
2. **Given** un proyecto sin historias, **When** abro "Backlog", **Then** veo "Aún no hay historias" y
   el formulario para crear la primera.

---

### Edge Cases

- Los campos de criterio en blanco o con solo espacios se ignoran; si no queda ninguno con texto, se
  aplica el error de criterios de aceptación.
- El formulario siempre muestra al menos un campo de criterio; el último campo no se puede quitar.
- "Agregar criterio" y "Quitar" no guardan nada ni muestran mensajes de éxito; solo vuelven a mostrar
  el formulario con los datos escritos.
- Presionar Enter dentro de un campo guarda la historia; nunca agrega ni quita criterios.
- La historia siempre pertenece al proyecto activo del servidor; un identificador de proyecto
  enviado en el formulario se ignora.
- Si la selección de proyecto se pierde al guardar, los datos del formulario no se conservan; se pide
  elegir el proyecto de nuevo (limitación aceptada en HU-20b).
- Una historia sin Story Points o sin horas estimadas se muestra "Sin estimar" en esa columna y no
  como cero.
- Un título o una descripción con HTML o scripts se muestra como texto (regla de HU-20a/HU-20b).
- Los títulos de historias pueden repetirse: HU-02 no define unicidad; no hay escenario propio.
- Las horas estimadas aceptan coma o punto como separador decimal y se muestran con coma; no se
  admiten separadores de miles. 9999,99 es válido; 10000 y 9999,995 no.
- Un valor de horas con más de dos decimales se rechaza y no se redondea en silencio.
- La lista no tiene paginación; con muchas historias se muestra completa. Su orden es provisional
  (las más recientes primero) hasta que HU-13 defina el orden por prioridad.
- La pantalla no muestra la descripción ni los criterios de cada historia; verlos y editarlos es de
  HU-13.
- Recargar la página después de guardar no crea una segunda historia ni repite el mensaje (patrón de
  HU-20b; se prueba en T016).
- Si falla la lectura de las historias, se muestra la página de error de la aplicación y no un
  formulario incompleto (HU-20a; se prueba en T037).
- Título y descripción vacíos, prioridad inválida, Story Points inválidos y horas inválidas o por
  encima del máximo se prueban por separado, una prueba por regla.
- Las cookies de contexto son de sesión y no expiran por tiempo: el caso real es que no haya
  proyecto activo válido (sesión nueva, proyecto eliminado o cookie inválida); se prueba en T029.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer el acceso al formulario de alta desde la sección "Backlog" del
  menú, que requiere un proyecto activo.
- **FR-002**: El formulario MUST pedir título, descripción, prioridad, uno o más criterios de
  aceptación, Story Points (opcional) y horas estimadas (opcional).
- **FR-003**: Un alta válida MUST crear la historia según las reglas de HU-02, con estado inicial
  "Pendiente", en el proyecto activo, mostrar "Historia creada correctamente" una sola vez y mostrar
  la historia en la lista.
- **FR-004**: "Agregar criterio" MUST volver a mostrar el formulario con un campo de criterio más,
  conservando lo escrito y sin guardar. Esto MUST funcionar sin JavaScript.
- **FR-005**: "Quitar" MUST eliminar ese campo de criterio, conservando lo escrito y sin guardar, y
  MUST NOT quitar el último campo restante.
- **FR-006**: El sistema MUST ignorar los criterios en blanco al guardar; si no queda ninguno con
  texto, MUST aplicar el error de criterios de aceptación.
- **FR-007**: Cuando falten criterios de aceptación, el sistema MUST mostrar el error de HU-02 junto
  al campo, conservar los datos y MUST NOT crear la historia.
- **FR-008**: Cuando falten los Story Points, el sistema MUST crear la historia y MUST mostrar una
  sola advertencia que combina la confirmación y el aviso de que falta la estimación.
- **FR-009**: Cuando falten las horas estimadas, el sistema MUST crear la historia sin advertencia y
  MUST mostrarlas como "Sin estimar".
- **FR-010**: Ante un título o descripción vacíos o con solo espacios, el sistema MUST mostrar el
  mensaje de HU-02 junto a cada campo, conservar los datos y MUST NOT crear la historia.
- **FR-011**: Ante una prioridad inválida, el sistema MUST mostrar el mensaje de HU-02 junto al campo,
  conservar los datos y MUST NOT crear la historia.
- **FR-012**: Ante Story Points que no son un valor válido, el sistema MUST mostrar un mensaje junto
  al campo, conservar los datos y MUST NOT crear la historia.
- **FR-013**: Ante horas estimadas negativas, cero, no numéricas o con más de dos decimales, el
  sistema MUST mostrar "El valor de horas no es válido" junto al campo, conservar
  los datos, MUST NOT crear la historia y MUST NOT redondear en silencio.
- **FR-014**: Ante horas estimadas mayores que 9999,99, el sistema MUST mostrar el mensaje del máximo
  junto al campo, conservar los datos y MUST NOT crear la historia.
- **FR-015**: El sistema MUST aceptar la coma y el punto como separador decimal de las horas
  estimadas, guardar el valor ingresado sin alterarlo y mostrarlo con coma.
- **FR-016**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo completarse",
  conservar los datos, MUST NOT mostrar éxito y MUST NOT mostrar la historia en la lista.
- **FR-017**: Cuando no haya un proyecto activo válido al guardar, el sistema MUST bloquear el
  guardado y redirigir a la selección de proyecto con el mensaje de HU-20b.
- **FR-018**: El proyecto de la historia MUST salir siempre del contexto del servidor y MUST NOT
  tomarse de un campo del formulario.
- **FR-019**: La lista MUST mostrar identificador, título, prioridad, Story Points, horas estimadas
  ("Sin estimar" si falta alguno) y estado de las historias del proyecto activo, MUST NOT mostrar
  historias de otros proyectos y MUST salir con las más recientes primero como orden provisional.
- **FR-020**: Cuando el proyecto no tenga historias, el sistema MUST mostrar "Aún no hay historias" y
  el formulario de alta.
- **FR-021**: Si falla la lectura de las historias, el sistema MUST mostrar la página de error de la
  aplicación y MUST NOT mostrar un formulario incompleto.
- **FR-022**: Recargar la página después de guardar MUST NOT crear una segunda historia ni repetir el
  mensaje.
- **FR-023**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: permitir cargar historias en el Backlog del proyecto activo y ver la lista desde una
pantalla web, con las reglas de HU-02 y sin que la persona pierda lo que escribió ante un
error.

**Entradas** (formulario `POST /backlog`):

| Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|
| `titulo` | texto | sí | Vacío o solo espacios se trata como vacío; sin límite de largo |
| `descripcion` | texto | sí | Igual que el título |
| `prioridad` | `Alta`, `Media` o `Baja` | sí | Selector; otro valor solo llega con un envío manipulado |
| `story_points` | 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89 | no | Selector; vacío es válido |
| `horas_estimadas` | número positivo | no | Hasta dos decimales, máximo 9999,99, coma o punto; vacío es válido |
| `criterio` | texto (un campo por criterio) | sí, al menos uno con texto | Los vacíos se ignoran |
| `accion` | `guardar`, `agregar_criterio` o `quitar_criterio:N` | no | Por defecto `guardar` |

**Salidas esperadas**:

- Alta válida: redirección 303 a `/backlog`, mensaje de una sola vista y la historia en la lista con
  estado "Pendiente".
- Agregar o quitar un criterio: respuesta 200 con el formulario, sin guardar.
- Datos inválidos: respuesta 422 con el formulario, los datos conservados y un mensaje junto a cada
  campo con error.
- Falla de guardado: respuesta 500 con el formulario, los datos conservados y "La operación no pudo
  completarse".
- Falla de lectura: página de error de HU-20a.
- Lista: identificador, título, prioridad, Story Points, horas estimadas y estado.

**Reglas de negocio**:

- Las reglas de título, descripción, prioridad, criterios, Story Points y horas estimadas son las de
  HU-02, con los límites de horas que se agregan al dominio (ver Assumptions); las aplica el
  servicio, no el handler. El handler solo arma el
  envío (acciones, criterios en blanco, números) y traduce los errores a campos.
- Una historia nueva siempre queda "Pendiente" y pertenece al proyecto activo del servidor.
- Un valor vacío de Story Points o de horas no es un error: se crea y se muestra "Sin estimar".
- Solo falta de Story Points genera advertencia.

**Restricciones**:

- Sin migraciones ni dependencias nuevas; no se modifican HU-20a, HU-20b ni HU-21.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Campo | Mensaje | Respuesta |
|---|---|---|---|
| Título vacío o solo espacios | `titulo` | El de HU-02 | 422 |
| Descripción vacía o solo espacios | `descripcion` | El de HU-02 | 422 |
| Prioridad fuera de Alta, Media o Baja | `prioridad` | El de HU-02 | 422 |
| Ningún criterio con texto | `criterio` | El de HU-02 | 422 |
| Story Points fuera de la lista | `story_points` | El de HU-02 si lo define; si no, "Los Story Points no son un valor válido" | 422 |
| Horas negativas, cero, no numéricas o con más de dos decimales | `horas_estimadas` | El de HU-02 si lo define; si no, "El valor de horas no es válido" (el mismo texto de HU-03 y HU-17) | 422 |
| Horas mayores que 9999,99 | `horas_estimadas` | "Las horas estimadas no pueden superar 9999,99" | 422 |
| Falla de guardado | general | "La operación no pudo completarse" | 500 |
| Falla de lectura | página | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | redirección | Mensaje de HU-20b | 303 |

Los mensajes de HU-02 se toman del código (T001). Los textos propios de HU-22
("Historia creada correctamente", el mensaje combinado de Story Points faltantes, el de Story Points
inválidos y el del máximo de horas) son una propuesta del equipo y se definen en un solo lugar
(`backlog_form.go`).

### Key Entities *(include if feature involves data)*

- **Historia del Backlog**: Definida por HU-02; esta historia la crea desde el formulario y la lista.
- **Criterio de aceptación**: Texto de una condición de la historia; una historia tiene uno o más.
- **Fila del Backlog**: Identificador, título, prioridad, Story Points, horas estimadas y estado de
  una historia.
- **Proyecto activo**: Proyecto elegido en HU-20b sobre el que trabaja la pantalla.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las altas válidas aparece en el Backlog del proyecto activo con estado
  "Pendiente".
- **SC-002**: El 100% de las entradas inválidas, incluidas las horas estimadas, se rechaza con el
  mensaje junto a su campo, conservando los datos y sin crear historias.
- **SC-003**: El 100% de las historias guardadas sin Story Points se crea con la advertencia de
  estimación faltante, y el 0% de las guardadas sin horas muestra advertencia.
- **SC-004**: El 100% de las historias queda en el proyecto activo y 0 en otro proyecto, verificado
  con dos proyectos.
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin éxito
  y sin la historia en el Backlog.
- **SC-006**: El 100% de los guardados sin proyecto activo válido se bloquea y redirige a la selección
  de proyecto sin guardar nada.
- **SC-007**: La pantalla responde en menos de 500 ms bajo condiciones normales de uso (la prueba de
  T035 la mide con la base local y menos de 50 historias).
- **SC-008**: El 100% de los valores de horas entre 0,01 y 9999,99 con hasta dos decimales se guarda
  y se muestra tal como se ingresó, sin redondeo.

## Assumptions

- El modelo, las validaciones, el caso de uso de alta y la persistencia de historias son los de
  HU-02; esta historia no cambia sus reglas ni sus mensajes. Los valores válidos son los de la
  migración `002`: prioridad `Alta`, `Media` o `Baja`; Story Points 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89.
- Las horas estimadas ya existen en el alta de HU-02 (columna `estimated_hours` de la migración
  `002`, mayor que 0). Si el código de HU-02 todavía no rechaza más de dos decimales ni valores
  mayores que 9999,99, ni acepta la coma, esos límites se agregan en el dominio, en el mismo lugar,
  con su prueba (T001 y T018), y se avisa a quien tome HU-17 para que los reutilice. HU-22 no depende
  de que HU-17 esté integrada.
- El máximo de 9999,99 es el de la columna `NUMERIC(6,2)`; es una propuesta del equipo y debe figurar
  también en HU-17.
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; la selección del proyecto activo y la redirección sin contexto, de HU-20b.
- Ordenar el Backlog por prioridad, filtrar por estado y editar historias son de HU-13; esta lista es
  mínima y HU-13 la extiende y reemplaza su orden provisional.
- Asignar historias a un Sprint es de HU-23; esta pantalla no lo ofrece. Los estados "En Sprint",
  "En progreso" y "Hecho" se muestran tal cual los guarda cada historia.
- No hay roles: cualquier persona con un proyecto activo puede cargar historias.
- Los criterios dinámicos funcionan con envíos del formulario (sin JavaScript); la interfaz no
  depende de scripts.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests de handler y unitarios | BDD | Código |
|---|---|---|---|---|
| 1. Alta de historia | FR-001 a FR-003, FR-018 | T009, T010 | T017 | `backlog_form.go`, `backlog_handlers.go`, `backlog.html` |
| 2. Agregar criterio | FR-004 | T009, T011 | T017 | `backlog_form.go`, `backlog_handlers.go` |
| 3. Quitar criterio | FR-005 | T009, T011 | T017 | `backlog_form.go`, `backlog_handlers.go` |
| 4. No se puede quitar el último criterio | FR-005 | T009, T011 | T017 | `backlog_form.go` |
| 5. Criterios en blanco | FR-006 | T009, T012 | T017 | `backlog_form.go` |
| 6. Sin Story Points | FR-008 | T013 | T017 | `backlog_form.go`, `backlog_handlers.go` |
| 7. Sin horas estimadas | FR-009 | T014 | T017 | `backlog_form.go`, `backlog.html` |
| 8. Horas con coma decimal | FR-015 | T009, T015 | T017 | `backlog_form.go` |
| 9. Sin criterios de aceptación | FR-007 | T022 | T030 | `backlog_form.go` |
| 10. Título vacío | FR-010 | T023 | T030 | `backlog_form.go` |
| 11. Descripción vacía | FR-010 | T023 | T030 | `backlog_form.go` |
| 12. Prioridad inválida | FR-011 | T024, T033 | T030 | `backlog_form.go` |
| 13. Story Points inválidos | FR-012 | T025, T033 | T030 | `backlog_form.go` |
| 14. Horas estimadas inválidas | FR-013 | T026 | T030 | `backlog_form.go` |
| 15. Horas por encima del máximo | FR-014 | T027 | T030 | `backlog_form.go` |
| 16. Error al guardar la historia | FR-016, FR-023 | T028, T034 | T030 | `backlog_handlers.go` |
| 17. Lista del Backlog | FR-019 | T006, T035 | T038 | `postgres_story_catalog.go`, `backlog.html` |
| 18. Backlog vacío | FR-020 | T036 | T038 | `backlog.html` |
| (HU-20b) Recarga sin duplicar | FR-022 | T016 | | `backlog_handlers.go` |
| (HU-20b) Sin proyecto activo al guardar | FR-017, FR-018 | T029 | | `backlog_handlers.go` |
| (HU-20a) Error al leer | FR-021 | T037 | | `backlog_handlers.go` |
