# Feature Specification: Crear y asignar Sprint

**Feature Branch**: `003-create-sprint`

**Created**: 2026-09-30

**Status**: Draft

**Input**: User description: "HU-04 (Issue #5): Sistema para que el Scrum Master cree un Sprint definiendo su Sprint Goal y fechas, y le asigne historias del Product Backlog, con el objetivo de enmarcar el trabajo de la iteración."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crear Sprint y asignar historias (Priority: P1)

Como Scrum Master, quiero crear un Sprint con su objetivo y período, y asignarle historias
pendientes del Product Backlog para delimitar el trabajo que el equipo realizará durante
la iteración.

**Why this priority**: La asignación de historias a un Sprint transforma el Product Backlog
en un plan de trabajo concreto para el seguimiento de la iteración.

**Independent Test**: Puede probarse con un proyecto que tenga historias pendientes,
creando el Sprint 1, definiendo su objetivo y fechas válidas, asignando tres historias y
verificando que quedan asociadas y con estado "En Sprint".

**Acceptance Scenarios**:

1. **Given** un proyecto con historias en el Product Backlog, **When** el Scrum Master
   crea un Sprint 1 con su "Sprint Goal" y le asigna tres historias pendientes, **Then**
   el sistema asocia esas historias al Sprint y actualiza su estado a "En Sprint" para el
   seguimiento.
2. **Given** una historia ya pertenece a un Sprint activo, **When** el Scrum Master intenta
  asignarla también a otro Sprint, **Then** el sistema bloquea la acción y muestra un mensaje
  indicando que la historia ya está asignada a otro Sprint en curso.
3. **Given** una historia pertenece a un Sprint activo, **When** el Scrum Master primero la
  quita de ese Sprint y luego la asigna a otro, **Then** el sistema permite la nueva asignación
  sin conflicto.
4. **Given** el Scrum Master está creando un nuevo Sprint, **When** completa el formulario,
  **Then** el sistema asigna automáticamente un identificador secuencial (Sprint 1, Sprint 2,
  etc.) que no puede editarse.
5. **Given** el Scrum Master está creando un nuevo Sprint, **When** intenta guardarlo sin
  completar el campo de Sprint Goal, **Then** el sistema impide la creación y solicita
  completar el objetivo del Sprint.

---

### User Story 2 - Validar el período del Sprint (Priority: P2)

Como Scrum Master, quiero que el sistema rechace períodos inválidos para evitar que un
Sprint tenga una duración nula o una fecha de finalización anterior a su inicio.

**Why this priority**: La consistencia de las fechas es necesaria para ordenar la ejecución
de la iteración y calcular correctamente su seguimiento.

**Independent Test**: Puede probarse intentando guardar un Sprint con fecha de fin anterior
o igual a la fecha de inicio y verificando que no se crea y se informa el error.

**Acceptance Scenarios**:

1. **Given** el Scrum Master está creando un nuevo Sprint, **When** ingresa una fecha de
   fin anterior o igual a la fecha de inicio, **Then** el sistema impide la creación y
   muestra un mensaje de error indicando la inconsistencia temporal.

---

### User Story 3 - Crear Sprint sin historias (Priority: P3)

Como Scrum Master, quiero poder guardar un Sprint aunque todavía no tenga historias
asignadas para registrar la estructura de la iteración y completar su planificación luego.

**Why this priority**: El equipo puede definir el objetivo y el período del Sprint antes
de terminar la selección de historias, sin bloquear la planificación inicial.

**Independent Test**: Puede probarse completando el objetivo y fechas válidas sin seleccionar
historias, guardando el Sprint y verificando que se crea con una advertencia visible.

**Acceptance Scenarios**:

1. **Given** el Scrum Master completó el "Sprint Goal" y las fechas del Sprint, **When**
   intenta guardarlo sin historias asignadas, **Then** el sistema permite la creación y
   muestra una advertencia de que el Sprint quedó sin historias asignadas.

---

### Edge Cases

- Si no hay un proyecto seleccionado, el sistema debe impedir la creación e indicar que se
  requiere un proyecto.
- El identificador del Sprint debe generarse automáticamente y no debe presentarse como un
  campo editable del formulario.
- Si el "Sprint Goal" está vacío, el sistema debe impedir la creación e indicar que es
  obligatorio.
- Si falta la fecha de inicio o la fecha de fin, el sistema debe impedir la creación e
  indicar cuál es el dato obligatorio faltante.
- Si se intenta asignar una historia que ya pertenece a otro Sprint activo, el sistema debe
  bloquear la acción y mostrar que la historia ya está asignada a otro Sprint en curso.
- Para mover una historia entre Sprints, primero debe quitarse del Sprint activo actual y
  luego asignarse al nuevo Sprint; no se permite el reemplazo directo.
- Si se seleccionan historias que no están en estado "Pendiente", el sistema debe impedir
  su asignación o excluirlas informando cuáles no pudieron asignarse.
- Si se asignan menos o más de tres historias, el sistema debe permitir la cantidad siempre
  que todas sean elegibles; tres historias es el caso de aceptación de referencia.
- Si el Sprint se guarda sin historias, debe crearse sin asociaciones y mostrar la advertencia
  sin tratar la situación como un error de validación.
- Si ocurre un error al guardar el Sprint o sus asociaciones, el sistema debe informar que
  la operación no se completó y evitar asociaciones parciales.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST permitir al Scrum Master iniciar la creación de un Sprint
  dentro de un proyecto seleccionado.
- **FR-002**: El sistema MUST generar automáticamente un identificador secuencial para cada
  Sprint, con el formato "Sprint 1", "Sprint 2" y sucesivos, y MUST impedir que el usuario
  lo edite.
- **FR-003**: El formulario MUST permitir ingresar el "Sprint Goal", la fecha de inicio y
  la fecha de finalización.
- **FR-004**: El sistema MUST validar que el Sprint Goal, la fecha de inicio y la fecha de
  finalización estén completos antes de guardar.
- **FR-005**: El sistema MUST validar que la fecha de finalización sea posterior a la fecha
  de inicio; no debe aceptar fechas iguales ni anteriores.
- **FR-006**: El sistema MUST permitir seleccionar historias del Product Backlog del
  proyecto para asignarlas al Sprint.
- **FR-007**: Al guardar un Sprint con historias pendientes seleccionadas, el sistema MUST
  asociar cada historia al Sprint y actualizar su estado a "En Sprint".
- **FR-008**: El sistema MUST permitir guardar un Sprint sin historias asignadas cuando el
  resto de los datos sea válido.
- **FR-009**: Al guardar un Sprint sin historias, el sistema MUST mostrar una advertencia
  explícita de que quedó sin historias asignadas, sin impedir la creación.
- **FR-010**: El sistema MUST impedir que una historia pertenezca simultáneamente a dos
  Sprints activos, bloquear el segundo intento de asignación e informar el conflicto.
- **FR-011**: Para mover una historia a otro Sprint, el sistema MUST permitir primero quitarla
  del Sprint activo actual y luego asignarla al nuevo Sprint, sin conservar la asociación
  anterior.
- **FR-012**: El sistema MUST impedir la asignación de historias que no estén disponibles
  para el Sprint e informar las historias rechazadas y el motivo.
- **FR-013**: Después de una creación exitosa, el sistema MUST mostrar el Sprint con su
  objetivo, fechas, historias asociadas y estado actualizado de cada historia.
- **FR-014**: Si falla la creación o asociación, el sistema MUST evitar estados parciales:
  no debe confirmar el Sprint como creado si sus asociaciones requeridas no se completaron.
- **FR-015**: Ante un error de validación, el sistema MUST conservar los datos ingresados
  para que el Scrum Master pueda corregirlos.

### Key Entities *(include if feature involves data)*

- **Sprint**: Iteración de trabajo de un proyecto, identificada por un identificador secuencial
  generado automáticamente, un Sprint Goal obligatorio, una fecha de inicio, una fecha de
  finalización y sus historias asociadas.
- **Sprint Goal**: Objetivo que orienta el trabajo comprometido para la iteración.
- **Historia de usuario**: Elemento del Product Backlog que puede estar pendiente o ser
  asignado a un Sprint.
- **Asignación de historia**: Relación exclusiva entre una historia y un Sprint activo que
  actualiza la historia al estado "En Sprint" para su seguimiento; se elimina antes de una
  nueva asignación cuando la historia se mueve de Sprint.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los Sprints creados con datos válidos y tres historias pendientes
  asocia las tres historias y las muestra con estado "En Sprint".
- **SC-002**: El 100% de los intentos con fecha de fin anterior o igual a la fecha de inicio
  es rechazado, no crea un Sprint y muestra el error de inconsistencia temporal.
- **SC-003**: El 100% de los Sprints válidos guardados sin historias se crea correctamente y
  muestra una advertencia visible de backlog vacío.
- **SC-004**: Al menos el 90% de los Scrum Masters de prueba puede crear un Sprint y asignar
  historias en el primer intento con datos válidos.
- **SC-005**: Un Scrum Master puede completar la creación de un Sprint y asignar tres
  historias en menos de 3 minutos cuando dispone de la información requerida.
- **SC-006**: El 100% de los Sprints creados recibe un identificador secuencial único que el
  usuario no puede editar.
- **SC-007**: El 100% de los intentos de asignar una historia que ya pertenece a otro Sprint
  activo es bloqueado y muestra un mensaje de conflicto.
- **SC-008**: El 100% de las historias quitadas de un Sprint activo puede asignarse a otro
  Sprint sin conflicto cuando cumple las demás reglas de elegibilidad.

## Assumptions

- El Scrum Master tiene permisos para crear Sprints y modificar las historias del proyecto.
- El proyecto ya existe y contiene un Product Backlog accesible, aunque puede no contener
  historias al momento de crear el Sprint.
- "En Sprint" es el estado de seguimiento asignado a una historia cuando se incorpora a un
  Sprint; los estados posteriores quedan fuera de esta historia.
- El sistema genera identificadores secuenciales comenzando por "Sprint 1" y garantiza que no
  sean editables por el usuario.
- El "Sprint Goal" es un campo de texto obligatorio e independiente del identificador del
  Sprint.
- Una historia puede pertenecer como máximo a un Sprint activo; para moverla se elimina
  primero la asignación actual y luego se crea la nueva.
- Un Sprint puede existir sin historias asignadas y estas pueden incorporarse posteriormente.
- La edición, cierre, eliminación y replanificación de Sprints quedan fuera del alcance de
  esta historia.
