# Quickstart: Crear y asignar Sprint

## Prerequisites

- Go 1.22 o posterior instalado.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Repositorio ubicado en su raíz.
- Paquete `src/sprint`, `PostgreSQLSprintStore`, integración con el modelo de backlog de HU-02
   y `go.mod` creados según el plan.
- Variables de conexión PostgreSQL configuradas según el adaptador de pruebas.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias del dominio de Sprint y
los escenarios de aceptación disponibles.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Creación y asignación exitosa:
   - proyecto existente;
   - Sprint Goal no vacío;
   - fecha final posterior a la inicial;
   - tres historias `Pendiente` pasan a `En Sprint`;
   - el Sprint recibe una etiqueta automática como `Sprint 1`.
2. Sprint Goal obligatorio:
   - Goal vacío rechaza la creación;
   - el error solicita completar el objetivo;
   - no se crea un Sprint.
3. Fechas inválidas:
   - fecha final anterior rechaza la creación;
   - fecha final igual rechaza la creación;
   - el error informa inconsistencia temporal.
4. Sprint sin historias:
   - el Sprint se crea con una colección vacía;
   - se devuelve una advertencia de Sprint sin historias;
   - la advertencia no se trata como error.
5. Exclusividad:
   - asignar una historia que ya está en otro Sprint activo se bloquea;
   - el mensaje identifica el conflicto;
   - no se cambia la asociación existente.
6. Movimiento:
   - quitar primero la historia del Sprint actual la devuelve a `Pendiente`;
   - asignarla después al nuevo Sprint la cambia a `En Sprint`;
   - la historia queda asociada solo al nuevo Sprint.
7. Integridad de la operación:
   - una asignación inválida no crea asociaciones parciales;
   - el contador secuencial no se consume si la creación falla antes de persistir.
8. PostgreSQL:
   - `PostgreSQLSprintStore` usa pgx mediante `database/sql`;
   - el Sprint, sus historias y estados se persisten en PostgreSQL;
   - una falla transaccional no deja un Sprint o asociaciones parciales;
   - la prueba de integración se ejecuta contra PostgreSQL real.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar las reglas reales del proyecto y del Sprint, no reemplazarlas con
mocks o métodos exclusivos de test.
