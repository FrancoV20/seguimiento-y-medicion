# Data Model: Registrar horas trabajadas

## Story

Historia del Product Backlog (HU-02) asignada a un Sprint (HU-04). Esta historia solo la consulta.

### Fields

- `id`: identificador de la historia.
- `projectID`: proyecto al que pertenece.
- `activeSprintID`: Sprint activo al que pertenece, opcional.

### Rules

- Solo una historia con `activeSprintID` apuntando a un Sprint `Activo` admite carga de esfuerzo.
- Esta historia no modifica el estado, la prioridad ni los Story Points de la historia.

## Sprint

Iteración definida por HU-04. Esta historia solo consulta su estado.

### Fields

- `id`: identificador único del Sprint.
- `status`: debe ser `Activo` para aceptar carga de esfuerzo.

## Member

Integrante definido por HU-01.

### Fields

- `id`: identificador del integrante.
- `displayName`: nombre de presentación.

### Rules

- El integrante del registro debe existir y pertenecer al proyecto de la historia.

## Hours

Valor de horas trabajadas con aritmética exacta.

### Representation

- Entero de centésimas de hora (`int64`); `4` horas se representan como `400`.
- Se guarda como `NUMERIC(5,2)` en PostgreSQL.

### Rules

- Debe ser mayor que 0 y menor o igual a 24 por registro.
- Se aceptan hasta dos decimales; más decimales, valores no numéricos o vacíos son inválidos.
- La suma de horas nunca usa punto flotante.

## EffortEntry

Registro individual de esfuerzo sobre una historia.

### Fields

- `id`: identificador único generado por el sistema.
- `storyID`: historia a la que pertenece el registro.
- `memberID`: integrante responsable.
- `workDate`: fecha de calendario del trabajo realizado.
- `activity`: descripción de la actividad, no vacía después de normalizar espacios.
- `hours`: horas trabajadas, en el rango válido.
- `createdAt`: momento en que se guardó el registro.

### Rules

- Un registro pertenece a una única historia y a un único integrante.
- Un registro guardado no se modifica en esta historia.
- `workDate` no puede ser posterior a la fecha actual.

## AccumulatedEffort

Esfuerzo real acumulado de una historia.

### Fields

- `storyID`: historia consultada.
- `totalHours`: suma de `hours` de todos sus `EffortEntry`.

### Rules

- `totalHours = sum(hours)` de los registros de la historia; no se almacena como campo mutable.
- Una historia sin registros tiene `totalHours = 0`.
- HU-07 lee este valor como `actualHours` de cada historia terminada.

## LogEffortRequest

Datos ingresados por el Product Builder.

### Fields

- `storyID`: historia seleccionada.
- `memberID`: integrante responsable.
- `workDate`: fecha del trabajo.
- `activity`: actividad realizada.
- `hours`: horas ingresadas como texto o número, antes de validar.

El identificador y la fecha de creación del registro no forman parte de la solicitud.

## LogEffortResult

Resultado de registrar el esfuerzo.

### Fields

- `entry`: registro creado cuando la operación tiene éxito.
- `accumulatedHours`: esfuerzo real acumulado actualizado de la historia.
- `successMessage`: confirmación posterior a la persistencia.
- `validationErrors`: errores de campos, uno por campo inválido.
- `storageError`: mensaje genérico "La operación no pudo completarse", sin detalles internos.

### Invariants

- Éxito implica un único registro nuevo y un acumulado incrementado exactamente en sus horas.
- Un error de validación o de almacenamiento no crea registros ni modifica el acumulado.
- Un error de almacenamiento nunca produce `successMessage`.

### Messages

- Horas inválidas: "El valor de horas no es válido".
- Actividad vacía: "La descripción de la actividad es obligatoria".
- Fecha omitida: "La fecha es requerida".
- Integrante no válido: "Se debe especificar el integrante responsable".
- Historia sin Sprint activo: "La historia no está asignada a un Sprint activo".
- Falla de almacenamiento: "La operación no pudo completarse".

## EffortStore

Interfaz interna para validar el contexto, guardar registros y leer el acumulado.

### Operations

- `StoryAcceptsEffort(storyID)`: indica si la historia existe y pertenece a un Sprint activo.
- `MemberBelongsToStoryProject(memberID, storyID)`: verifica que el integrante exista y pertenezca al proyecto de la historia.
- `SaveEntry(entry)`: guarda el registro y devuelve el acumulado actualizado dentro de una única operación.
- `AccumulatedHours(storyID)`: devuelve la suma de horas de la historia.

### Rules

- `SaveEntry` debe confirmar el registro y el cálculo del acumulado como una unidad.
- Los errores de almacenamiento no producen mensaje de éxito ni mutaciones parciales.

## PostgreSQLEffortStore

Implementación concreta de `EffortStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` como driver de PostgreSQL.
- `github.com/jackc/pgx/v5/stdlib` para registrar pgx y operar mediante `database/sql`.

### Rules

- Debe persistir cada `EffortEntry` en una tabla `effort_logs` con referencias a la historia (migración 002) y al integrante (migración 001).
- Debe tener una restricción `CHECK (hours > 0 AND hours <= 24)`.
- Debe ejecutar `SaveEntry` dentro de una transacción de `database/sql` con `ROLLBACK` ante cualquier error.
- Debe consultar el estado `Activo` del Sprint de la historia (migración 003).
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse al sistema.

### State Transition

`Solicitud -> validación -> EffortStore.SaveEntry (transacción) -> Registro guardado + acumulado actualizado`.

Si falla la validación o el almacenamiento, el registro no se considera creado.
