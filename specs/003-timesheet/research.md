# Research: Registrar horas trabajadas

## Decision 1: Servicio de dominio Go con persistencia PostgreSQL

- **Decision**: Implementar las reglas de registro en un servicio Go separado de `PostgreSQLEffortStore`, que usará `github.com/jackc/pgx/v5` mediante `database/sql` para validar el contexto de la historia, guardar el registro y leer el acumulado.
- **Rationale**: La constitución v1.1.0 exige PostgreSQL y una implementación concreta para toda interfaz de almacenamiento antes de integrar la feature.
- **Alternatives considered**: Guardar solo en memoria o dejar la persistencia para después. Se descartan porque no cumplen el principio VI.

## Decision 2: Acumulado derivado de los registros

- **Decision**: El esfuerzo real acumulado de una historia se calcula como la suma de las horas de sus registros (`SUM(hours)`) y no se guarda como un campo mutable en la historia.
- **Rationale**: Evita desincronización entre registros y total, conserva la trazabilidad exigida por el curso (Integrante, Fecha, Actividad, Horas) y no obliga a modificar las tablas de HU-02 ni HU-04. HU-07 lee el mismo valor como `actualHours`.
- **Alternatives considered**: Columna `actual_hours` en la tabla de historias actualizada con cada carga. Se descarta porque duplica el dato, invade el esquema de otra historia y puede quedar inconsistente.

## Decision 3: Horas como valor decimal exacto de hasta dos decimales

- **Decision**: Representar las horas en centésimas de hora con un entero (`int64`) en un tipo `Hours`, interpretar la entrada con hasta dos decimales y guardar `NUMERIC(5,2)` en PostgreSQL.
- **Rationale**: Evita errores de punto flotante al sumar (0,1 + 0,2) sin agregar dependencias externas y mantiene coherencia con la aritmética decimal controlada de HU-07.
- **Alternatives considered**: `float64`, que acumula errores de redondeo, o una librería decimal externa, que agrega una dependencia innecesaria para esta historia.

## Decision 4: Rango válido de horas por registro

- **Decision**: Aceptar horas `> 0` y `<= 24`; rechazar negativas, cero, mayores a 24, no numéricas y con más de dos decimales con el mensaje "El valor de horas no es válido". Además, una restricción `CHECK` de la tabla impide guardar valores fuera de rango.
- **Rationale**: Es el criterio de aceptación del Issue #4. La restricción en la base protege los datos aunque otra función escriba directamente en la tabla.
- **Alternatives considered**: Validar solo en el servicio. Se descarta porque dejaría datos inválidos posibles ante un bug o una escritura directa.

## Decision 5: Validación completa y acumulación de errores

- **Decision**: Validar horas, actividad, fecha, integrante y estado de la historia antes de tocar la base y devolver todos los errores juntos.
- **Rationale**: El usuario corrige todo de una vez y se garantiza que un registro inválido nunca llega al almacenamiento. Es el mismo criterio que usa HU-07 al informar todas las historias inválidas.
- **Alternatives considered**: Detenerse en el primer error. Se descarta porque obliga a reintentar varias veces.

## Decision 6: Historia elegible = asignada a un Sprint activo

- **Decision**: Una historia admite carga si su `activeSprintID` (definido en HU-04) apunta a un Sprint con estado `Activo`.
- **Rationale**: HU-04 asigna historias a Sprints y no a personas; es la única noción de "asignada" que existe en el modelo actual y coincide con el escenario del Issue ("historia asignada en el Sprint activo").
- **Alternatives considered**: Exigir un responsable individual por historia. Se descarta porque ninguna historia existente lo define y ampliaría el alcance.

## Decision 7: Integrante válido del proyecto

- **Decision**: El integrante debe existir y pertenecer al proyecto al que pertenece la historia (HU-01 define los integrantes del proyecto).
- **Rationale**: Garantiza la trazabilidad del esfuerzo y evita cargas atribuidas a personas ajenas al proyecto.
- **Alternatives considered**: Aceptar cualquier texto como nombre de integrante. Se descarta porque permite datos inconsistentes y duplicados.

## Decision 8: Fecha obligatoria y no futura

- **Decision**: La fecha es obligatoria, se interpreta como fecha de calendario y no puede ser posterior a la fecha actual. El servicio recibe un reloj inyectable para que las pruebas sean deterministas.
- **Rationale**: Permite cargar trabajo de días anteriores sin aceptar esfuerzo "adelantado". La inyección del reloj evita pruebas dependientes del día de ejecución.
- **Alternatives considered**: Aceptar solo la fecha actual, que impide corregir olvidos, o aceptar cualquier fecha, que permite datos sin sentido. Esta decisión está pendiente de confirmación del equipo.

## Decision 9: Atomicidad del registro

- **Decision**: Insertar el registro y calcular el acumulado dentro de una misma transacción de `database/sql`; ante cualquier error se hace `ROLLBACK`, se informa "La operación no pudo completarse" y no se devuelve mensaje de éxito.
- **Rationale**: Cumple el escenario de error al guardar y evita registros parciales o acumulados que no reflejan la base.
- **Alternatives considered**: Insertar y luego consultar en operaciones separadas. Se descarta porque una falla intermedia podría informar éxito falso.

## Decision 10: Migraciones con golang-migrate

- **Decision**: Crear `db/migrations/005_create_effort_logs.up.sql` y `005_create_effort_logs.down.sql`, dependientes de las migraciones 001 (proyectos e integrantes), 002 (historias) y 003 (Sprints).
- **Rationale**: `golang-migrate` exige pares `.up.sql`/`.down.sql` por versión. La migración 005 debe aplicarse antes de las pruebas de integración de HU-07, que leen los registros de esfuerzo, aunque su número sea posterior a 004.
- **Alternatives considered**: Un único archivo `.sql` sin par `.down`, que `migrate` no reconoce.

## Decision 11: Pruebas unitarias, BDD e integración

- **Decision**: Usar `go test ./...`, `testing` y tablas de casos para validación de horas; escenarios BDD por cada criterio del Issue #4; y pruebas de integración contra PostgreSQL real para `PostgreSQLEffortStore`.
- **Rationale**: Mantiene el ciclo TDD rápido y verifica el contrato real de almacenamiento, incluyendo la restricción `CHECK` y la transacción.
- **Alternatives considered**: Solo pruebas unitarias con un store en memoria. Se descartan porque no validan el motor real.
