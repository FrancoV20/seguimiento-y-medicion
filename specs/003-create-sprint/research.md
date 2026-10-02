# Research: Crear y asignar Sprint

## Decision 1: Servicio Go con persistencia PostgreSQL

- **Decision**: Implementar creación, asignación y movimiento en un paquete Go con reglas de dominio separadas de `PostgreSQLSprintStore`, que usará `github.com/jackc/pgx/v5` mediante `database/sql`.
- **Rationale**: La constitución v1.1.0 fija PostgreSQL como persistencia obligatoria y exige una implementación concreta de toda interfaz de almacenamiento antes de integrar la feature.
- **Alternatives considered**: Mantener solo un dominio en memoria o empezar por una pantalla/endpoint. Se descartan porque no cumplen el gate de persistencia real.

## Decision 2: Identificador secuencial generado por proyecto

- **Decision**: Al crear un Sprint, el proyecto calcula el siguiente número disponible y genera `Sprint N`; ese identificador no forma parte de la entrada del usuario.
- **Rationale**: Cumple la decisión confirmada y evita colisiones o nombres manuales. El número se determina como máximo existente más uno, comenzando por 1.
- **Alternatives considered**: UUID visible o nombre editable. Se descartan porque no cumplen el formato secuencial solicitado.

## Decision 3: Sprint Goal obligatorio e independiente

- **Decision**: El Sprint Goal se normaliza quitando espacios externos y se rechaza si queda vacío; se guarda como texto separado del identificador.
- **Rationale**: El Goal representa el objetivo de la iteración y no debe confundirse con el identificador automático.
- **Alternatives considered**: Usar el identificador como objetivo o permitir Goal vacío. Se descartan porque contradicen FR-003/FR-004.

## Decision 4: Período estrictamente válido

- **Decision**: La fecha de finalización debe ser posterior a la fecha de inicio; igualdad y fechas anteriores se rechazan.
- **Rationale**: Es la regla explícita de aceptación y evita Sprints de duración nula o negativa.
- **Alternatives considered**: Permitir un Sprint de un solo día con fechas iguales. Se descarta porque la especificación exige `fin > inicio`.

## Decision 5: Estado inicial activo

- **Decision**: Un Sprint creado queda `Activo`, porque puede recibir historias posteriormente y la exclusividad aplica a Sprints activos.
- **Rationale**: Permite guardar Sprints sin historias y define cuándo una historia queda bloqueada para otra asignación.
- **Alternatives considered**: Crear en estado borrador. Se descarta porque no está definido por la historia y dejaría ambigua la regla de exclusividad.

## Decision 6: Asignación exclusiva y movimiento en dos pasos

- **Decision**: Antes de asociar una historia, el servicio verifica que no pertenezca a otro Sprint activo. Para moverla, una operación explícita elimina la asociación anterior y devuelve la historia a `Pendiente`; luego una nueva asignación la pasa a `En Sprint`.
- **Rationale**: Cumple la regla dura del equipo y hace observable el flujo de movimiento sin reemplazo directo.
- **Alternatives considered**: Reasignación automática o permitir dos asociaciones. Se descartan porque ocultarían el conflicto o romperían el seguimiento.

## Decision 7: Sprints sin historias son válidos

- **Decision**: La colección de historias puede estar vacía; la creación tiene éxito y devuelve una advertencia no bloqueante.
- **Rationale**: La planificación puede registrar primero el objetivo y período y asignar historias después.
- **Alternatives considered**: Exigir al menos una historia. Se descarta porque contradice el escenario confirmado.

## Decision 8: Validación antes de mutación persistida

- **Decision**: Validar Goal, fechas, disponibilidad de historias y exclusividad antes de crear el Sprint o actualizar estados; `PostgreSQLSprintStore` ejecutará la creación y asociaciones dentro de una transacción de `database/sql`.
- **Rationale**: Protege la consistencia del proyecto y cumple FR-014/FR-015 sin dejar relaciones parciales en PostgreSQL.
- **Alternatives considered**: Crear el Sprint y asignar historias progresivamente. Se descarta porque puede dejar relaciones incompletas.

## Decision 9: Adaptador PostgreSQL antes de integrar

- **Decision**: `PostgreSQLSprintStore` debe implementarse y probarse contra PostgreSQL antes de integrar el servicio de Sprint.
- **Rationale**: La interfaz de almacenamiento no puede quedar como abstracción sin implementación concreta bajo el principio VI.
- **Alternatives considered**: Integrar primero un store en memoria y agregar PostgreSQL después. Se descarta porque contradice la constitución.

## Decision 10: Pruebas unitarias y de integración

- **Decision**: Usar `go test ./...`, `testing` y tablas de casos para el dominio, más pruebas de integración PostgreSQL para `PostgreSQLSprintStore` usando `database/sql` y pgx.
- **Rationale**: Las pruebas unitarias mantienen el ciclo TDD rápido y las de integración verifican transacciones, asociaciones exclusivas y persistencia real.
- **Alternatives considered**: Solo pruebas unitarias con mocks o memoria. Se descartan porque no validarían el contrato PostgreSQL requerido.
