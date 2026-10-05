# Quickstart: Agregar historia al Product Backlog

## Prerequisites

- Go 1.27.1 o posterior instalado.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada.
- Repositorio ubicado en su raíz.
- Implementación del paquete `src/backlog`, `PostgreSQLBacklogStore` y `go.mod` creados según
   el plan.
- Variable `DATABASE_URL` configurada para PostgreSQL, por ejemplo:
   `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Aplicar las migraciones pendientes en orden numérico. La migración 002 depende de que la 001,
que crea los proyectos, ya esté aplicada:

```powershell
migrate -path db/migrations -database "$DATABASE_URL" up
```

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias del backlog y los
escenarios de aceptación disponibles.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Proyecto activo y datos válidos:
   - se crea una única historia;
   - queda asociada al proyecto correcto;
   - inicia en estado `Pendiente`;
   - se muestra la confirmación.
2. Sin criterios de aceptación:
   - el alta se rechaza;
   - se solicita agregar al menos un criterio;
   - no se agrega una historia al backlog.
3. Sin Story Points:
   - el alta se permite;
   - la historia conserva ausencia de estimación;
   - se muestra la alerta correspondiente.
4. Story Points informados:
   - valores `1, 2, 3, 5, 8, 13, 21, 34, 55, 89` se aceptan;
   - valores fuera de la serie se rechazan;
   - un rechazo no agrega ni duplica la historia.
5. Campos obligatorios y contexto:
   - proyecto inactivo o inexistente se rechaza;
   - título, descripción o prioridad vacíos se rechazan;
   - criterios vacíos o compuestos solo por espacios se rechazan.
6. PostgreSQL:
   - `PostgreSQLBacklogStore` usa pgx mediante `database/sql`;
   - la historia y su asociación al proyecto se persisten en PostgreSQL;
   - una falla transaccional no deja una historia parcialmente registrada;
   - la prueba de integración se ejecuta contra PostgreSQL real.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar las reglas reales del agregado del backlog y no sustituirlas con
mocks o métodos exclusivos de test.
