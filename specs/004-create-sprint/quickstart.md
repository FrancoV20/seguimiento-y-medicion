# Quickstart: Crear y asignar Sprint

## Prerequisites

- Go 1.27.1 instalado en Windows desde [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi); verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada; seguir la instalación de Windows documentada en [README.md](../../README.md).
- Repositorio ubicado en su raíz.
- Paquete `src/sprint`, `PostgreSQLSprintStore`, integración con el modelo de backlog de HU-02
   y `go.mod` creados según el plan.
- Variable `DATABASE_URL` configurada para PostgreSQL, por ejemplo:
   `postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable`.
   El puerto `5433` es el del `docker-compose.yml` por defecto; si tu `docker-compose.override.yml`
   usa otro usuario, clave o puerto, ajustá la URL (ver "Base de datos" en el [README.md](../../README.md)).

## Instalar golang-migrate

Si `migrate -version` no responde en tu terminal, instalarlo desde PowerShell:

```powershell
go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest
$env:Path += ";$(go env GOPATH)\bin"
migrate -version
```

Si muestra `dev`, es válido: el binario funciona aunque no informe un número de release.
La segunda línea agrega la carpeta de herramientas de Go al `PATH` solo de la terminal actual;
para dejarlo permanente, agregar `$(go env GOPATH)\bin` a la variable de entorno `Path` del
usuario y abrir una nueva terminal.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

## Aplicar las migraciones

Las migraciones se aplican siempre en orden numérico. La migración de Sprints
(`004_create_sprints`) depende de que la `001_create_projects`, que crea los proyectos, y la
`002_create_backlog_stories`, que crea las historias del backlog, ya estén aplicadas, porque los
Sprints referencian proyectos e historias existentes. Si alguna falta en tu rama, traerla desde la
rama de su dueño siguiendo la sección C del [README.md](../../README.md).

Cada migración tiene un par de archivos en `db/migrations/`: `NNN_nombre.up.sql` (aplica) y
`NNN_nombre.down.sql` (revierte).

1. Definir la URL de conexión (dura mientras la ventana de PowerShell esté abierta):

   ```powershell
   $env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
   ```

2. Ver qué versión está aplicada (`no migration` significa base vacía):

   ```powershell
   migrate -path db/migrations -database $env:DATABASE_URL version
   ```

3. Aplicar todas las migraciones pendientes, en orden numérico:

   ```powershell
   migrate -path db/migrations -database $env:DATABASE_URL up
   ```

4. Para revertir la última migración (por ejemplo, si cambió el archivo de un compañero):

   ```powershell
   migrate -path db/migrations -database $env:DATABASE_URL down 1
   ```

Si `migrate` queda en estado "dirty", avisar al equipo antes de usar `force`.

## Ejecutar las pruebas

Con las migraciones aplicadas, ejecutar:

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
