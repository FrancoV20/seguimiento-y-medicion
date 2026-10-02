# Feature Specification: Agregar historia al Product Backlog

**Feature Branch**: `002-add-backlog-story`

**Created**: 2026-09-30

**Status**: Draft

**Input**: User description: "HU-02 (Issue #3): Sistema para que el Scrum Master agregue una historia de usuario al Product Backlog definiendo su título, descripción, prioridad, estado y Story Points, con el objetivo de planificar el trabajo del equipo."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar historia en el backlog (Priority: P1)

Como Scrum Master, quiero agregar una historia de usuario a un proyecto activo con su
título, descripción, prioridad, criterios de aceptación y Story Points para organizar
el trabajo que el equipo deberá planificar.

**Why this priority**: El Product Backlog necesita historias concretas para representar
el trabajo del proyecto y permitir su posterior priorización y planificación.

**Independent Test**: Puede probarse seleccionando un proyecto activo, completando los
datos válidos de una historia y verificando que aparece en su Product Backlog con estado
inicial "Pendiente".

**Acceptance Scenarios**:

1. **Given** el Scrum Master seleccionó un proyecto activo, **When** ingresa el título,
   la descripción, la prioridad y al menos un criterio de aceptación, define Story Points
   cuando corresponda y guarda, **Then** la historia se añade al Product Backlog del
   proyecto con el estado inicial "Pendiente".
2. **Given** el Scrum Master está creando una historia, **When** guarda sin haber definido
  Story Points, **Then** el sistema permite la creación igual, pero muestra una alerta
  advirtiendo que falta completar la estimación.

---

### User Story 2 - Exigir criterios de aceptación (Priority: P2)

Como Scrum Master, quiero que cada historia tenga al menos un criterio de aceptación
antes de guardarla para asegurar que el resultado esperado quede especificado.

**Why this priority**: Los criterios de aceptación hacen verificable la historia y evitan
que el backlog contenga trabajo ambiguo o difícil de validar.

**Independent Test**: Puede probarse intentando guardar una historia sin criterios de
aceptación y verificando que no se registra y que el sistema solicita agregarlos.

**Acceptance Scenarios**:

1. **Given** se está redactando una nueva historia, **When** se intenta guardar sin
   especificar al menos un criterio de aceptación, **Then** el sistema solicita que se
   agreguen y no incorpora la historia al Product Backlog.

---

### Edge Cases

- Si no hay un proyecto activo seleccionado, el sistema debe impedir el registro e indicar
  que primero se debe seleccionar un proyecto.
- Si el título está vacío, el sistema debe impedir el registro e indicar que es obligatorio.
- Si la descripción está vacía, el sistema debe impedir el registro e indicar que es
  obligatoria.
- Si no se define una prioridad, el sistema debe impedir el registro e indicar que es
  obligatoria.
- Si los criterios de aceptación contienen solo espacios en blanco, deben considerarse
  ausentes y el sistema debe solicitar al menos uno válido.
- Si no se informan Story Points, el sistema debe conservar la historia sin estimación,
  permitir la creación y mostrar una alerta advirtiendo que falta completar la estimación.
- Si se informan Story Points, el valor debe pertenecer a la serie de Fibonacci admitida
  por el sistema; un valor fuera de esa serie debe rechazarse e indicar el error.
- Toda historia creada mediante esta funcionalidad debe comenzar con estado "Pendiente";
  el formulario no debe permitir asignarle otro estado inicial.
- Si ocurre un error al guardar, el sistema debe informar que no pudo completarse la
  operación y no debe mostrar la historia como incorporada al backlog.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST permitir al Scrum Master iniciar la creación de una historia
  desde el Product Backlog de un proyecto activo.
- **FR-002**: El sistema MUST requerir un proyecto activo antes de permitir guardar la
  historia.
- **FR-003**: El formulario MUST permitir ingresar el título, la descripción, la prioridad,
  los criterios de aceptación y los Story Points de la historia.
- **FR-004**: El sistema MUST validar que el título, la descripción y la prioridad estén
  completos antes de guardar.
- **FR-005**: El sistema MUST requerir al menos un criterio de aceptación con contenido no
  vacío antes de registrar la historia.
- **FR-006**: Si los datos obligatorios son válidos, el sistema MUST agregar la historia al
  Product Backlog del proyecto seleccionado.
- **FR-007**: Toda historia creada mediante esta funcionalidad MUST registrarse con estado
  inicial "Pendiente".
- **FR-008**: El sistema MUST permitir registrar Story Points cuando el Scrum Master los
  defina únicamente mediante un valor de la serie de Fibonacci admitida, y MUST representar
  explícitamente la ausencia de estimación cuando no se informen.
- **FR-009**: Si no se informan Story Points, el sistema MUST permitir el registro cuando el
  resto de los datos obligatorios sea válido y MUST mostrar una alerta indicando que falta
  completar la estimación.
- **FR-010**: Si se informa un valor de Story Points que no pertenece a la serie de Fibonacci
  admitida, el sistema MUST impedir el registro, indicar el error y conservar los demás datos
  ingresados.
- **FR-011**: Si falta un dato obligatorio o no hay criterios de aceptación, el sistema MUST
  impedir el registro, indicar qué debe corregirse y conservar los demás datos ingresados.
- **FR-012**: Después de un registro exitoso, el sistema MUST mostrar una confirmación y la
  historia MUST ser visible dentro del Product Backlog del proyecto.
- **FR-013**: El sistema MUST evitar duplicar la historia cuando el usuario reintenta después
  de un error de validación o de persistencia.

### Key Entities *(include if feature involves data)*

- **Historia de usuario**: Unidad de trabajo del Product Backlog con título, descripción,
  prioridad, criterios de aceptación, Story Points y estado.
- **Product Backlog**: Conjunto ordenado de historias perteneciente a un proyecto activo.
- **Proyecto activo**: Proyecto seleccionado y disponible para recibir nuevas historias.
- **Criterio de aceptación**: Condición verificable que define cuándo una historia satisface
  el comportamiento esperado; cada historia debe tener al menos uno.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los intentos con proyecto activo y datos obligatorios válidos agrega
  una única historia al Product Backlog con estado "Pendiente".
- **SC-002**: El 100% de los intentos sin criterios de aceptación es rechazado, solicita su
  incorporación y no agrega una historia al backlog.
- **SC-003**: Al menos el 90% de los Scrum Masters de prueba puede registrar una historia
  válida en el primer intento.
- **SC-004**: Un Scrum Master puede completar y guardar una historia en menos de 3 minutos
  cuando dispone de la información requerida.
- **SC-005**: Toda historia visible en el Product Backlog conserva el proyecto al que fue
  asociada, su prioridad, sus criterios de aceptación y su estimación de Story Points,
  cuando esta fue informada.
- **SC-006**: El 100% de las historias guardadas sin Story Points muestra una alerta de
  estimación pendiente y permanece disponible para completarla posteriormente.
- **SC-007**: El 100% de los valores de Story Points aceptados pertenece a la serie de
  Fibonacci definida por el sistema; los valores fuera de ella son rechazados.

## Assumptions

- Solo un Scrum Master con permisos sobre el proyecto puede utilizar esta funcionalidad.
- El proyecto seleccionado ya fue creado y se encuentra activo, como resultado de HU-01.
- La prioridad se expresa mediante un conjunto ordenado de valores definido por el sistema;
  esta historia no define los nombres exactos de esos valores.
- Story Points es opcional al crear la historia; si no se informa, la historia se registra
  sin estimación y muestra una alerta para completarla posteriormente.
- Cuando se informa Story Points, el valor debe seleccionarse de la serie de Fibonacci
  admitida por el sistema y no puede ingresarse como un número libre.
- El estado "Pendiente" es el estado inicial común para toda historia recién creada.
- La edición, eliminación, ordenamiento avanzado y movimiento de historias entre estados
  quedan fuera del alcance de esta historia.
