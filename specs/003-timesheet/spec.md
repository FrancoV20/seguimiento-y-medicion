# Feature Specification: Registrar horas trabajadas

**Feature Branch**: `003-timesheet`

**Created**: 2026-10-09

**Status**: Draft

**Input**: User description: "HU-03 (Issue #4): Como Product Builder, quiero registrar la fecha, actividad realizada y las horas trabajadas en una historia de usuario asignada, para poder comparar posteriormente el esfuerzo estimado con el real."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Registrar esfuerzo diario en una historia asignada (Priority: P1)

Como Product Builder, quiero registrar la fecha, la actividad realizada y las horas trabajadas
en una historia de mi Sprint activo para que el sistema acumule el esfuerzo real de esa historia.

**Why this priority**: Es el núcleo de la historia: sin horas reales acumuladas no se puede
comparar el esfuerzo estimado con el real ni calcular la desviación del Sprint (HU-07).

**Independent Test**: Puede probarse con una historia asignada a un Sprint activo, registrando
4 horas con fecha actual y la actividad "Desarrollo de endpoint en Go", y verificando que el
esfuerzo real acumulado de la historia aumenta exactamente 4 horas.

**Acceptance Scenarios**:

1. **Given** un Product Builder con una historia asignada en el Sprint activo, **When** ingresa
   la fecha actual, detalla la tarea técnica (por ejemplo "Desarrollo de endpoint en Go") y
   carga "4" horas, **Then** el sistema suma esas 4 horas al esfuerzo real acumulado de esa
   historia de usuario.
2. **Given** una historia que ya tiene horas acumuladas, **When** se registra un nuevo esfuerzo
   válido sobre ella, **Then** el acumulado resultante es la suma de todos los registros
   válidos de esa historia, sin sobrescribir los anteriores.

---

### User Story 2 - Rechazar cantidades de horas inválidas (Priority: P2)

Como Product Builder, quiero que el sistema rechace cantidades de horas que no tienen sentido
para que el esfuerzo real acumulado sea confiable.

**Why this priority**: Horas negativas, nulas o excesivas distorsionan la comparación entre
esfuerzo estimado y real y arruinan las métricas del Sprint.

**Independent Test**: Puede probarse intentando registrar -1, 0 y 25 horas, verificando que
cada intento se rechaza con el mensaje de valor de horas no válido y que el acumulado de la
historia no cambia.

**Acceptance Scenarios**:

1. **Given** un Product Builder que está registrando esfuerzo en una historia, **When** ingresa
   una cantidad de horas negativa, igual a cero, o mayor a 24 en un mismo registro, **Then** el
   sistema rechaza la carga y muestra un mensaje indicando que el valor de horas no es válido.

---

### User Story 3 - Exigir actividad, fecha e integrante (Priority: P2)

Como Product Builder, quiero que el sistema exija la actividad realizada, la fecha y el
integrante responsable para que cada registro de esfuerzo sea trazable.

**Why this priority**: El requerimiento del curso exige registrar Integrante, Fecha, Actividad
realizada y Horas trabajadas; un registro sin alguno de ellos no sirve para auditar el esfuerzo.

**Independent Test**: Puede probarse enviando por separado un registro sin actividad, otro sin
fecha y otro sin integrante válido, verificando que cada uno se rechaza con su mensaje y que no
se guarda nada.

**Acceptance Scenarios**:

1. **Given** un Product Builder que intenta registrar esfuerzo en una historia asignada,
   **When** deja el campo de "Actividad realizada" vacío y presiona guardar, **Then** el sistema
   impide el registro y muestra un mensaje indicando que la descripción de la actividad es
   obligatoria.
2. **Given** un Product Builder que intenta registrar el esfuerzo, **When** omite ingresar la
   "Fecha" y presiona guardar, **Then** el sistema rechaza la carga y muestra un mensaje
   indicando que la fecha es requerida.
3. **Given** que se está intentando registrar el esfuerzo de una tarea, **When** no se asocia
   un "Integrante" válido al registro de horas, **Then** el sistema bloquea la acción indicando
   que se debe especificar el integrante responsable.

---

### User Story 4 - Informar fallas al guardar sin falso éxito (Priority: P3)

Como Product Builder, quiero que el sistema me avise cuando el registro no pudo guardarse para
no creer que mis horas quedaron cargadas cuando no es así.

**Why this priority**: Un falso mensaje de éxito haría perder horas trabajadas y falsearía las
métricas del Sprint.

**Independent Test**: Puede probarse simulando una falla del almacenamiento durante un registro
con todos los campos válidos, verificando que se informa que la operación no pudo completarse,
que no hay mensaje de éxito y que el acumulado de la historia no cambia.

**Acceptance Scenarios**:

1. **Given** un Product Builder que completó todos los campos obligatorios correctamente,
   **When** ocurre un error al intentar guardar el registro en la base de datos, **Then** el
   sistema informa que la operación no pudo completarse y no muestra un mensaje de éxito.

---

### Edge Cases

- Si la historia no está asignada a un Sprint activo (sin Sprint, o con un Sprint que ya no
  está activo), el sistema debe rechazar el registro e informar que la historia no admite
  carga de esfuerzo.
- Si la cantidad de horas es exactamente 24, el registro es válido; si es 24,01 o más, es
  inválido.
- Si la cantidad de horas es mayor que 0 pero menor que 1 (por ejemplo 0,5), el registro es
  válido.
- Si la cantidad de horas tiene más de dos decimales, el sistema debe rechazarla como valor de
  horas no válido en lugar de redondearla en silencio.
- Si el valor de horas no es numérico o está vacío, el sistema debe rechazarlo como valor de
  horas no válido.
- Si la actividad contiene solo espacios en blanco, se considera vacía.
- Si faltan varios campos a la vez, el sistema debe informar todos los errores en una sola
  respuesta y no guardar nada.
- Si el integrante no existe o no pertenece al proyecto de la historia, se considera no válido.
- Si la fecha es posterior a la fecha actual, el sistema debe rechazar el registro.
- Un mismo integrante puede registrar varias cargas sobre la misma historia en el mismo día; cada
  una se acumula como un registro independiente.
- Una carga rechazada o fallida no debe modificar el esfuerzo real acumulado de la historia.
- El esfuerzo acumulado debe ser el mismo al volver a consultarlo mientras no se registren nuevas
  cargas.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST permitir registrar un esfuerzo con integrante, fecha, actividad
  realizada y horas trabajadas sobre una historia asignada a un Sprint activo.
- **FR-002**: El sistema MUST sumar las horas de cada registro válido al esfuerzo real
  acumulado de la historia y mostrar el acumulado actualizado.
- **FR-003**: El sistema MUST conservar cada registro de esfuerzo como un dato independiente con
  su integrante, fecha, actividad y horas, de modo que el acumulado de una historia sea la suma
  de todos sus registros válidos.
- **FR-004**: El sistema MUST aceptar únicamente horas mayores que 0 y menores o iguales a 24
  por registro, con hasta dos decimales.
- **FR-005**: Cuando las horas sean negativas, iguales a cero, mayores a 24, no numéricas o con
  más de dos decimales, el sistema MUST rechazar la carga y mostrar un mensaje indicando que el
  valor de horas no es válido.
- **FR-006**: El sistema MUST exigir una actividad realizada con contenido no vacío después de
  normalizar espacios; si falta, MUST rechazar la carga e indicar que la descripción de la
  actividad es obligatoria.
- **FR-007**: El sistema MUST exigir la fecha del registro; si falta, MUST rechazar la carga e
  indicar que la fecha es requerida.
- **FR-008**: El sistema MUST rechazar un registro con fecha posterior a la fecha actual.
- **FR-009**: El sistema MUST exigir un integrante válido, existente y perteneciente al proyecto
  de la historia; si falta o no es válido, MUST bloquear la acción e indicar que se debe
  especificar el integrante responsable.
- **FR-010**: El sistema MUST validar todos los campos antes de guardar, informar todos los
  errores de validación juntos y no guardar nada si existe al menos un error.
- **FR-011**: El sistema MUST rechazar el registro cuando la historia no esté asignada a un
  Sprint activo e informar el motivo.
- **FR-012**: Si ocurre una falla al guardar, el sistema MUST informar que la operación no pudo
  completarse, MUST NOT mostrar un mensaje de éxito y MUST NOT modificar el esfuerzo acumulado.
- **FR-013**: Los mensajes de error de almacenamiento MUST NOT exponer detalles internos de la
  base de datos.
- **FR-014**: El esfuerzo real acumulado de cada historia MUST estar disponible para consulta
  por otras funciones, en particular el cálculo de métricas del Sprint (HU-07), sin alterar los
  registros originales.
- **FR-015**: El resultado de acumular horas MUST ser reproducible: con los mismos registros
  válidos, el sistema MUST producir el mismo esfuerzo acumulado.

### Key Entities *(include if feature involves data)*

- **Registro de esfuerzo**: Carga individual de trabajo con integrante, fecha, actividad
  realizada y horas trabajadas, asociada a una única historia.
- **Esfuerzo real acumulado**: Suma de las horas de todos los registros válidos de una historia.
- **Historia asignada**: Historia del Product Backlog que pertenece a un Sprint activo y por lo
  tanto admite carga de esfuerzo.
- **Integrante**: Miembro del proyecto responsable del trabajo registrado.
- **Sprint activo**: Iteración en curso cuyas historias admiten carga de esfuerzo.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los registros válidos aumenta el esfuerzo real acumulado de la historia
  exactamente en las horas ingresadas.
- **SC-002**: El 100% de los registros con horas negativas, iguales a cero o mayores a 24 es
  rechazado con el mensaje de valor de horas no válido y sin modificar el acumulado.
- **SC-003**: El 100% de los registros sin actividad, sin fecha o sin integrante válido es
  rechazado con el mensaje correspondiente al campo faltante y sin guardar datos.
- **SC-004**: El 100% de las fallas de almacenamiento informa que la operación no pudo
  completarse, sin mensaje de éxito y sin registros parciales.
- **SC-005**: El esfuerzo acumulado se muestra actualizado en menos de 2 segundos después de
  guardar un registro, bajo condiciones normales de uso.
- **SC-006**: En el 100% de los casos, el esfuerzo acumulado de una historia coincide con la
  suma de sus registros guardados.
- **SC-007**: Al menos el 90% de los Product Builders de prueba puede registrar un esfuerzo
  válido sin ayuda en el primer intento.

## Assumptions

- Una historia "asignada" es una historia que pertenece a un Sprint activo, según la asociación
  definida en HU-04; esta historia no define un responsable individual por historia.
- El integrante del registro se selecciona entre los integrantes del proyecto de la historia
  (HU-01); no es necesario que sea quien tiene "asignada" la historia, porque HU-04 asigna
  historias a Sprints y no a personas. *(Pendiente de confirmación del equipo.)*
- Las horas se expresan en horas decimales con hasta dos decimales (por ejemplo 1,5 o 0,25).
  *(Pendiente de confirmación del equipo.)*
- Las horas máximas por registro son 24, según el criterio de aceptación; no se limita el total
  diario por integrante.
- La fecha del registro puede ser la fecha actual o una anterior, para cargar trabajo olvidado,
  pero no una fecha futura. *(Pendiente de confirmación del equipo.)*
- Cada carga es un registro nuevo; varias cargas el mismo día sobre la misma historia se
  acumulan.
- Editar o eliminar registros ya guardados queda fuera del alcance de esta historia.
- El esfuerzo estimado no se define aquí; esta historia solo provee las horas reales que HU-07
  utiliza para calcular la desviación.
- La interfaz gráfica, los reportes de esfuerzo por integrante y la exportación quedan fuera
  del alcance de esta historia.
