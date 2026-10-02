# Research: Crear nuevo proyecto

## Decision 1: Caso de uso Go con persistencia PostgreSQL detrás de una interfaz

- **Decision**: Implementar la validación y creación en un servicio de dominio Go que dependa de `ProjectStore`, con una implementación concreta PostgreSQL usando `github.com/jackc/pgx/v5` a través de `database/sql`.
- **Rationale**: La decisión del equipo fija PostgreSQL como motor y pgx como driver. La interfaz mantiene separadas las reglas de negocio de la persistencia, mientras que el adaptador concreto cumple la obligación de integración real.
- **Alternatives considered**: Acceso directo desde el servicio al driver o usar otro motor/driver. Se descartan porque acoplarían el dominio a infraestructura y contradicen la decisión tecnológica.

## Decision 2: Proyecto creado en estado activo

- **Decision**: Un proyecto creado correctamente comienza en estado `Activo`.
- **Rationale**: HU-02 exige seleccionar un proyecto activo para agregar historias; el alta de HU-01 debe producir un contexto utilizable por las features siguientes.
- **Alternatives considered**: Estado `Borrador` o sin estado. Se descartan porque bloquearían la continuidad del flujo del producto.

## Decision 3: Fechas inclusivas a nivel de día

- **Decision**: Comparar fechas como fechas de calendario, sin componente horario; `fechaFin == fechaInicio` es válido y `fechaFin < fechaInicio` es inválido.
- **Rationale**: La especificación permite explícitamente fechas iguales y solo exige impedir una finalización anterior.
- **Alternatives considered**: Usar timestamps y exigir una hora de finalización posterior. Se descarta porque agrega precisión no solicitada y puede producir errores de zona horaria.

## Decision 4: Integrantes existentes y sin duplicados

- **Decision**: La solicitud recibe identificadores de integrantes existentes; debe contener al menos uno y no puede repetir el mismo integrante.
- **Rationale**: La historia indica que los integrantes están disponibles para asociar y una colección sin duplicados representa correctamente la participación del proyecto.
- **Alternatives considered**: Crear integrantes como parte de esta operación. Se descarta porque su alta no pertenece a HU-01.

## Decision 5: Nombre normalizado y obligatorio

- **Decision**: Recortar espacios externos del nombre y rechazarlo si queda vacío. No se agrega una regla de unicidad o longitud máxima.
- **Rationale**: Cumple la regla de nombre no vacío sin inventar restricciones de negocio no definidas.
- **Alternatives considered**: Forzar nombres únicos o una longitud máxima. Se dejan para una decisión futura porque no aparecen en la especificación.

## Decision 6: Validación completa antes de persistir

- **Decision**: Validar todos los campos y el período antes de llamar a `ProjectStore.Create`; informar errores sin mutar el almacenamiento.
- **Rationale**: Evita proyectos incompletos y garantiza que el mensaje de éxito solo aparezca después de una persistencia confirmada.
- **Alternatives considered**: Guardar por etapas. Se descarta porque podría dejar proyectos parciales.

## Decision 7: Fallo de persistencia sin éxito ni duplicación

- **Decision**: Si el almacenamiento falla, el servicio devuelve un error de operación y no confirma el proyecto; los reintentos generan una sola creación por solicitud válida.
- **Rationale**: Cumple FR-006 y FR-009 y preserva la confianza del usuario en el mensaje de éxito.
- **Alternatives considered**: Mostrar éxito optimista antes de confirmar la base de datos. Se descarta porque contradice la aceptación.

## Decision 8: Implementación concreta antes de integrar

- **Decision**: `ProjectStore` no se considerará listo para integrarse hasta contar con el adaptador PostgreSQL mediante `database/sql` y pgx, incluyendo pruebas de integración contra PostgreSQL.
- **Rationale**: La constitución exige que toda interfaz de almacenamiento tenga una implementación concreta contra PostgreSQL antes de la integración.
- **Alternatives considered**: Integrar primero una implementación en memoria y dejar PostgreSQL para después. Se descarta porque solo sirve para pruebas unitarias y no satisface el gate de integración.

## Decision 9: Pruebas unitarias y de integración

- **Decision**: Usar `go test ./...`, `testing` y tablas de casos; las reglas se prueban unitariamente con un store controlado y el adaptador se verifica con integración PostgreSQL.
- **Rationale**: Se separa la velocidad de las pruebas de dominio de la verificación necesaria contra el motor real, sin introducir un framework BDD externo.
- **Alternatives considered**: Solo pruebas unitarias con un store en memoria. Se descartan porque no verificarían el contrato PostgreSQL requerido.
