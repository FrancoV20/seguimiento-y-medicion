# Feature Specification: Crear nuevo proyecto

**Feature Branch**: `001-create-project`

**Created**: 2026-09-30

**Status**: Draft

**Input**: User description: "HU-01 (Issue #2): Sistema para crear un nuevo proyecto ingresando su nombre, integrantes, fecha de inicio y finalización, con el objetivo de inicializar la gestión del proyecto en el sistema."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar un proyecto (Priority: P1)

Como usuario de la pantalla de Gestión de Proyectos, quiero ingresar el nombre,
los integrantes y las fechas de inicio y finalización para registrar un nuevo proyecto
y comenzar a gestionarlo en el sistema.

**Why this priority**: La creación del proyecto es el punto de entrada para organizar
sus integrantes, planificación, seguimiento y medición posterior.

**Independent Test**: Puede probarse completando todos los campos obligatorios con datos
válidos y verificando que el proyecto queda registrado y que el usuario recibe confirmación.

**Acceptance Scenarios**:

1. **Given** el usuario se encuentra en la pantalla de "Gestión de Proyectos",
   **When** completa el nombre, los integrantes, la fecha de inicio y la fecha de fin,
   y presiona "Crear", **Then** el sistema registra el proyecto en la base de datos
   y muestra un mensaje de éxito.

---

### User Story 2 - Evitar fechas inconsistentes (Priority: P2)

Como usuario que intenta registrar un proyecto, quiero recibir una advertencia cuando
la fecha de finalización sea anterior a la fecha de inicio para corregir los datos antes
de guardar información inválida.

**Why this priority**: La consistencia temporal es necesaria para que el seguimiento,
la planificación y las métricas del proyecto sean confiables.

**Independent Test**: Puede probarse ingresando una fecha de finalización anterior a la
fecha de inicio y verificando que no se crea ningún proyecto y se informa el error.

**Acceptance Scenarios**:

1. **Given** el usuario intenta crear un proyecto, **When** ingresa una fecha de
   finalización anterior a la fecha de inicio, **Then** el sistema impide la creación
   y muestra un mensaje de error indicando la inconsistencia temporal.

---

### Edge Cases

- Si el nombre está vacío, el sistema debe impedir la creación e indicar que es obligatorio.
- Si no se seleccionan integrantes, el sistema debe impedir la creación e indicar que se
  requiere al menos un integrante.
- Si falta la fecha de inicio o la fecha de fin, el sistema debe impedir la creación e
  indicar cuál es el dato obligatorio faltante.
- Si la fecha de finalización coincide con la fecha de inicio, el sistema debe permitir la
  creación, ya que no existe una finalización anterior al inicio.
- Si ocurre un error al registrar el proyecto, el sistema debe informar que no pudo
  completarse la operación y no debe mostrar un mensaje de éxito.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST permitir acceder a un formulario de creación desde la pantalla
  de "Gestión de Proyectos".
- **FR-002**: El formulario MUST solicitar el nombre del proyecto, al menos un integrante,
  la fecha de inicio y la fecha de finalización.
- **FR-003**: El sistema MUST validar que todos los campos obligatorios estén completos antes
  de intentar registrar el proyecto.
- **FR-004**: El sistema MUST validar que la fecha de finalización sea igual o posterior a la
  fecha de inicio.
- **FR-005**: Si los datos son válidos y el usuario presiona "Crear", el sistema MUST
  registrar un nuevo proyecto con el nombre, los integrantes y ambas fechas.
- **FR-006**: Después de un registro exitoso, el sistema MUST mostrar un mensaje de éxito al
  usuario.
- **FR-007**: Si la fecha de finalización es anterior a la fecha de inicio, el sistema MUST
  impedir el registro y mostrar un mensaje de error que indique la inconsistencia temporal.
- **FR-008**: Cuando la validación falla, el sistema MUST conservar los datos ingresados para
  permitir su corrección, excepto los valores que no puedan conservarse por restricciones
  del formulario.
- **FR-009**: El sistema MUST evitar registrar un proyecto cuando se produce una validación
  fallida o un error durante la operación de registro.

### Key Entities *(include if feature involves data)*

- **Proyecto**: Unidad de gestión que representa un proyecto de software; incluye nombre,
  integrantes, fecha de inicio y fecha de finalización.
- **Integrante**: Persona asociada al proyecto y responsable de participar en su gestión.
- **Período del proyecto**: Intervalo definido por la fecha de inicio y la fecha de
  finalización, cuya consistencia requiere que el fin no sea anterior al inicio.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los intentos con nombre, integrantes y fechas válidas registra un
  proyecto y muestra un mensaje de éxito.
- **SC-002**: El 100% de los intentos con fecha de finalización anterior a la fecha de inicio
  es rechazado sin registrar un proyecto y muestra un mensaje de error temporal.
- **SC-003**: Un usuario puede completar y enviar el formulario de creación en menos de
  2 minutos cuando dispone de los datos requeridos.
- **SC-004**: Al menos el 90% de los usuarios de prueba puede completar correctamente la
  creación en el primer intento con datos válidos.

## Assumptions

- La persona que utiliza la pantalla tiene permisos para crear proyectos.
- Los integrantes que se pueden seleccionar ya existen en el sistema o están disponibles
  para ser asociados durante la creación.
- El nombre del proyecto debe contener al menos un carácter no vacío; las reglas de longitud
  máxima no forman parte de esta historia.
- La persistencia y la gestión de errores de la base de datos son responsabilidades del
  sistema y no requieren una acción adicional del usuario.
- El alcance de esta historia se limita a crear el proyecto; la edición, eliminación,
  consulta detallada y gestión posterior quedan fuera de alcance.
