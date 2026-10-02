# Research: Agregar historia al Product Backlog

## Decision 1: Núcleo Go con persistencia PostgreSQL

- **Decision**: Implementar el alta de historias en un paquete Go con reglas de dominio separadas de `PostgreSQLBacklogStore`, que usará `github.com/jackc/pgx/v5` mediante `database/sql`.
- **Rationale**: La constitución v1.1.0 fija PostgreSQL como persistencia obligatoria y exige una implementación concreta de cada interfaz de almacenamiento antes de integrar la feature.
- **Alternatives considered**: Mantener solo un núcleo en memoria o implementar primero una pantalla/endpoint. Se descartan porque no satisfacen el gate de persistencia PostgreSQL ni existe una capa de aplicación establecida.

## Decision 2: El agregado del backlog controla la creación

- **Decision**: El Product Backlog recibirá una solicitud de alta y validará proyecto activo, campos obligatorios, criterios, Story Points y estado inicial antes de agregar la historia.
- **Rationale**: Centralizar la operación evita que distintos consumidores creen historias con estados o reglas diferentes.
- **Alternatives considered**: Exponer un constructor público sin validación. Se descarta porque permitiría historias inválidas fuera del caso de uso.

## Decision 3: Story Points opcional con ausencia explícita

- **Decision**: Representar Story Points ausente como un valor opcional/nulo distinto de cero. La historia puede guardarse y debe devolver una alerta de estimación pendiente.
- **Rationale**: La decisión del equipo permite crear la historia sin estimación y evita confundir “sin estimar” con una estimación de cero.
- **Alternatives considered**: Usar cero como valor por defecto o bloquear el guardado. Se descartan porque pierden la diferencia semántica o contradicen la aceptación confirmada.

## Decision 4: Escala Fibonacci fija para valores informados

- **Decision**: Aceptar únicamente `1, 2, 3, 5, 8, 13, 21, 34, 55, 89` cuando se informen Story Points. La lista debe quedar centralizada en la regla de validación del dominio.
- **Rationale**: Es la escala Fibonacci habitual de Planning Poker y satisface la restricción de no permitir valores libres. La ausencia se maneja por separado.
- **Alternatives considered**: Aceptar cualquier entero positivo o incluir 0. Se descartan porque permiten valores fuera de la escala definida; la ausencia ya está representada por el valor opcional.

## Decision 5: Prioridad como valor ordenado

- **Decision**: Usar las prioridades `Alta`, `Media` y `Baja` como conjunto inicial ordenado, con `Alta` por encima de `Media` y `Baja`.
- **Rationale**: La especificación exige prioridad pero no define etiquetas. Este conjunto mínimo es comprensible para usuarios y suficiente para ordenar el backlog sin agregar complejidad.
- **Alternatives considered**: Prioridad numérica libre o una escala de cinco niveles. Se descartan porque dificultan validación y no aportan valor en esta historia.

## Decision 6: Estado inicial inmutable durante el alta

- **Decision**: Toda historia creada por este caso de uso comienza en `Pendiente`; el estado no forma parte de la entrada editable.
- **Rationale**: La aceptación exige estado inicial `Pendiente` y evita que el alta saltee el flujo de trabajo del backlog.
- **Alternatives considered**: Permitir que el usuario seleccione el estado inicial. Se descarta porque contradice FR-007.

## Decision 7: Operación atómica con PostgreSQL

- **Decision**: Validar toda la solicitud antes de mutar el backlog y persistir la historia y su asociación al proyecto dentro de una transacción de `database/sql` en `PostgreSQLBacklogStore`.
- **Rationale**: Cumple FR-009 y FR-013, evita historias parcialmente persistidas y deja el adaptador listo para integrarse contra el motor real.
- **Alternatives considered**: Agregar primero en memoria o persistir historia y asociación por separado. Se descartan porque podrían dejar estados parciales o duplicados.

## Decision 8: Adaptador concreto antes de integrar

- **Decision**: `PostgreSQLBacklogStore` debe implementarse y probarse contra PostgreSQL antes de integrar el caso de uso al sistema.
- **Rationale**: La interfaz de almacenamiento no puede quedar como abstracción sin implementación concreta bajo la constitución vigente.
- **Alternatives considered**: Integrar primero un store en memoria y agregar PostgreSQL después. Se descarta porque contradice el principio VI y las restricciones tecnológicas.

## Decision 9: Pruebas unitarias y de integración

- **Decision**: Usar `go test ./...`, `testing` y tablas de casos para el dominio, más pruebas de integración PostgreSQL para `PostgreSQLBacklogStore` usando `database/sql` y pgx.
- **Rationale**: Las pruebas unitarias mantienen rápido el ciclo TDD y las de integración verifican el contrato contra el motor real.
- **Alternatives considered**: Solo pruebas unitarias con mocks o un store en memoria. Se descartan porque no validarían la persistencia PostgreSQL requerida.
