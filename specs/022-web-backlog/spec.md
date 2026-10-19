# Feature Specification: Interfaz web del Product Backlog

**Feature Branch**: `HU-22`

**Created**: 2026-10-18

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

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cargar una historia desde el formulario web (Priority: P1)

Como Scrum Master, quiero completar título, descripción, prioridad, Story Points y criterios de
aceptación en un formulario y guardar la historia, para planificar el trabajo sin comandos.

**Why this priority**: Es el flujo principal de HU-02 hecho consumible: sin historias en el Backlog
no se pueden armar Sprints (HU-23) ni cargar esfuerzo (HU-24).

**Independent Test**: Con un proyecto activo, completar el formulario con datos válidos y verificar
que la historia aparece en el Backlog con estado "Pendiente"; agregar y quitar criterios, dejar
criterios en blanco, guardar sin Story Points y recargar la página.

**Acceptance Scenarios**:

1. **Given** un proyecto seleccionado, **When** completo título, descripción, prioridad y al menos
   un criterio de aceptación y guardo, **Then** veo el mensaje de éxito y la historia aparece en el
   Backlog con estado "Pendiente".
2. **Given** el formulario de historia, **When** presiono "Agregar criterio", **Then** el
   formulario se vuelve a mostrar con un campo de criterio más, conserva lo ya escrito y no se guarda
   nada; y puedo quitar los campos que no use.
3. **Given** que dejo campos de criterio en blanco, **When** guardo con al menos un criterio con
   texto, **Then** los campos en blanco se ignoran y la historia se guarda solo con los criterios
   escritos.
4. **Given** el formulario de historia, **When** guardo sin Story Points, **Then** la historia se
   crea igual y veo una advertencia de que falta completar la estimación.
5. **Given** que acabo de guardar una historia, **When** recargo la página, **Then** no se crea una
   segunda historia ni se repite el mensaje.
6. **Given** el formulario de historia, **When** ingreso horas estimadas válidas y guardo, **Then** la
   historia se crea con esas horas estimadas y las veo en el Backlog.

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

1. **Given** el formulario de historia, **When** guardo sin ningún criterio de aceptación con
   texto, **Then** veo el error junto al campo de criterios, los datos se conservan y no se crea la
   historia.
2. **Given** el formulario de historia, **When** dejo vacío el título o la descripción, o ingreso una
   prioridad que no es válida, **Then** veo un mensaje junto a cada campo, los datos se conservan y
   no se crea la historia.
3. **Given** que ingreso Story Points que no son un valor válido, **When** guardo, **Then** veo un
   mensaje junto al campo, los datos se conservan y no se crea la historia.
4. **Given** un formulario válido, **When** falla el guardado, **Then** veo "La operación no pudo
   completarse", los datos se conservan, no se muestra mensaje de éxito y la historia no aparece en
   el Backlog.
5. **Given** que no hay un proyecto activo válido (no hay selección, el proyecto dejó de existir o la
   selección es inválida), **When** presiono guardar, **Then** no se guarda nada y se me redirige a
   la selección de proyecto con el mensaje de HU-20b.
6. **Given** que ingreso horas estimadas negativas, iguales a cero, no numéricas o con más de dos
   decimales, **When** guardo, **Then** veo un mensaje junto al campo, los datos se conservan y no se
   crea la historia.

---

### User Story 3 - Ver el Backlog del proyecto activo (Priority: P2)

Como Scrum Master, quiero ver las historias del proyecto en la misma pantalla, para confirmar lo que
cargué y saber qué hay antes de agregar otra.

**Why this priority**: Permite verificar el resultado del alta y orienta a quien planifica, pero no
bloquea la carga.

**Independent Test**: Con dos proyectos con historias distintas, abrir el Backlog de cada uno y
verificar columnas, orden y aislamiento; con un proyecto sin historias, verificar el mensaje y el
formulario; simular una falla de lectura.

**Acceptance Scenarios**:

1. **Given** un proyecto seleccionado con historias, **When** abro "Backlog", **Then** veo cada
   historia con título, prioridad, Story Points, horas estimadas y estado, con las más recientes
   primero, y solo las
   del proyecto activo.
2. **Given** un proyecto sin historias, **When** abro "Backlog", **Then** veo "Aún no hay historias"
   y el formulario para crear la primera.
3. **Given** que falla la lectura de las historias, **When** abro la pantalla, **Then** veo la página
   de error de la aplicación y no un formulario incompleto.

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
- Las historias sin Story Points se muestran como "Sin estimar" y no como cero.
- Un título o una descripción con HTML o scripts se muestra como texto (regla de HU-20a/HU-20b).
- Los títulos de historias pueden repetirse: HU-02 no define unicidad.
- Las horas estimadas son opcionales; si faltan se muestran "Sin estimar" y no como cero.
- Un valor de horas estimadas con más de dos decimales se rechaza y no se redondea en silencio.
- La lista no tiene paginación; con muchas historias se muestra completa.
- Las cookies de contexto son de sesión y no expiran por tiempo: el caso real es que no haya
  proyecto activo válido (sesión nueva, proyecto eliminado o cookie inválida).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer el acceso al formulario de alta desde la sección "Backlog" del
  menú, que requiere un proyecto activo.
- **FR-002**: El formulario MUST pedir título, descripción, prioridad, Story Points (opcional), horas estimadas (opcional) y uno
  o más criterios de aceptación.
- **FR-003**: Un alta válida MUST crear la historia según las reglas de HU-02, con estado inicial
  "Pendiente", en el proyecto activo, mostrar el mensaje de éxito una sola vez y mostrar la historia
  en la lista.
- **FR-004**: "Agregar criterio" MUST volver a mostrar el formulario con un campo de criterio más,
  conservando lo escrito y sin guardar; "Quitar" MUST eliminar ese campo sin guardar y sin permitir
  quitar el último. Esto MUST funcionar sin JavaScript.
- **FR-005**: El sistema MUST ignorar los criterios en blanco; si no queda ninguno con texto, MUST
  aplicar el error de criterios de aceptación.
- **FR-006**: Cuando falten criterios de aceptación, el sistema MUST mostrar el error junto al campo,
  conservar los datos y MUST NOT crear la historia.
- **FR-007**: Cuando falten los Story Points, el sistema MUST crear la historia y MUST mostrar una
  advertencia de que falta completar la estimación.
- **FR-008**: Ante un título o descripción vacíos o una prioridad inválida, el sistema MUST mostrar
  los mensajes de HU-02 junto a cada campo, conservar los datos y MUST NOT crear la historia.
- **FR-009**: Ante Story Points que no son un valor válido, el sistema MUST mostrar un mensaje junto
  al campo y MUST NOT crear la historia.
- **FR-010**: Ante una falla de guardado, el sistema MUST mostrar "La operación no pudo completarse",
  conservar los datos, MUST NOT mostrar éxito y MUST NOT mostrar la historia en la lista.
- **FR-011**: Cuando no haya un proyecto activo válido al guardar, el sistema MUST bloquear el
  guardado y redirigir a la selección de proyecto con el mensaje de HU-20b.
- **FR-012**: El proyecto de la historia MUST salir siempre del contexto del servidor y MUST NOT
  tomarse de un campo del formulario.
- **FR-013**: La lista MUST mostrar título, prioridad, Story Points, horas estimadas (cada uno "Sin
  estimar" si falta) y estado de las historias del proyecto activo, con las más recientes primero, y MUST NOT mostrar historias de otros
  proyectos.
- **FR-014**: Cuando el proyecto no tenga historias, el sistema MUST mostrar "Aún no hay historias" y
  el formulario de alta.
- **FR-015**: Si falla la lectura de las historias, el sistema MUST mostrar la página de error de la
  aplicación y MUST NOT mostrar un formulario incompleto.
- **FR-016**: Recargar la página después de guardar MUST NOT crear una segunda historia.
- **FR-017**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.
- **FR-018**: El formulario MUST permitir cargar las horas estimadas de la historia como dato
  opcional, y la historia creada MUST conservarlas.
- **FR-019**: Ante horas estimadas que no son un valor válido, el sistema MUST mostrar un mensaje
  junto al campo, conservar los datos y MUST NOT crear la historia.

### Key Entities *(include if feature involves data)*

- **Historia del Backlog**: Definida por HU-02; esta historia la crea desde el formulario y la lista.
- **Criterio de aceptación**: Texto de una condición de la historia; una historia tiene uno o más.
- **Fila del Backlog**: Título, prioridad, Story Points, horas estimadas y estado de una historia.
- **Proyecto activo**: Proyecto elegido en HU-20b sobre el que trabaja la pantalla.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las altas válidas aparece en el Backlog del proyecto activo con estado
  "Pendiente".
- **SC-002**: El 100% de las entradas inválidas, incluidas las horas estimadas, se rechaza con el
  mensaje junto a su campo,
  conservando los datos y sin crear historias.
- **SC-003**: El 100% de las historias guardadas sin Story Points se crea con la advertencia de
  estimación faltante.
- **SC-004**: El 100% de las historias queda en el proyecto activo y 0 en otro proyecto, verificado
  con dos proyectos.
- **SC-005**: El 100% de las fallas de guardado muestra "La operación no pudo completarse", sin éxito
  y sin la historia en el Backlog.
- **SC-006**: El 100% de los guardados sin proyecto activo válido se bloquea y redirige a la selección
  de proyecto sin guardar nada.
- **SC-007**: La pantalla responde en menos de 500 ms bajo condiciones normales de uso.

## Assumptions

- El modelo, las validaciones, el caso de uso de alta y la persistencia de historias son los de
  HU-02; esta historia no cambia sus reglas ni sus mensajes. Los valores válidos son los de la
  migración `002`: prioridad `Alta`, `Media` o `Baja`; Story Points 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89.
- Mensajes, formularios y protección contra envíos ajenos vienen de HU-20b; la estructura común y los
  errores, de HU-20a; la selección del proyecto activo y la redirección sin contexto, de HU-20b.
- Ordenar el Backlog por prioridad, filtrar por estado y editar historias son de HU-13; esta lista es
  mínima y HU-13 la extiende.
- Las horas estimadas son un dato opcional que HU-02 ya admite al crear la historia (columna opcional
  de la migración `002`); el formulario las carga. Modificarlas después y sus reglas con Sprint
  finalizado son de HU-17, y no hay pantalla de edición en el Sprint 2. La regla de formato (mayor que
  0 y hasta dos decimales) es la de HU-02 y HU-17; se confirma en T001.
- Asignar historias a un Sprint es de HU-23; esta pantalla no lo ofrece.
- No hay roles: cualquier persona con un proyecto activo puede cargar historias.
- Los criterios dinámicos funcionan con envíos del formulario (sin JavaScript); la interfaz no
  depende de scripts.
