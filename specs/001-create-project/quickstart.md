# Quickstart: Crear nuevo proyecto

## Prerequisites

- Go 1.27.1 instalado en Windows desde [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi); verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada; seguir la instalación de Windows documentada en [README.md](../../README.md).
- Repositorio ubicado en su raíz.
- Paquete `src/project`, adaptador `PostgreSQLProjectStore` y `go.mod` creados según el plan.
- Variable `DATABASE_URL` configurada para PostgreSQL, por ejemplo:
   `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Aplicar las migraciones en orden numérico:

```powershell
migrate -path db/migrations -database "$DATABASE_URL" up
```

Las migraciones deben ejecutarse siempre en orden numérico para preservar la estructura de la
base de datos. Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias del dominio y los
escenarios de aceptación disponibles.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Creación exitosa:
   - nombre no vacío;
   - al menos un integrante disponible;
   - fecha de inicio y fin informadas;
   - fin posterior o igual al inicio;
   - `ProjectStore.Create` recibe el proyecto;
   - el resultado confirma éxito y estado `Activo`.
2. Fechas inválidas:
   - fecha de fin anterior rechaza la creación;
   - no se llama a `ProjectStore.Create`;
   - se muestra un error de inconsistencia temporal.
3. Fechas iguales:
   - fecha de inicio igual a fecha de fin permite la creación.
4. Campos obligatorios:
   - nombre vacío se rechaza;
   - integrantes ausentes se rechazan;
   - fecha de inicio o fin ausente se rechaza;
   - los datos restantes se conservan para corregir el formulario.
5. Persistencia:
   - un error de `ProjectStore` no muestra éxito;
   - no se confirma el proyecto como creado;
   - un reintento válido no genera duplicación dentro del flujo del caso de uso.
6. Integrantes:
   - un integrante inexistente se rechaza;
   - integrantes duplicados se rechazan con un error de validación explícito;
   - los integrantes duplicados no se normalizan ni se ignoran en silencio;
   - al menos un integrante válido queda asociado al proyecto.
7. PostgreSQL:
   - `PostgreSQLProjectStore` usa pgx mediante `database/sql`;
   - el proyecto y sus integrantes se persisten en PostgreSQL;
   - un fallo transaccional no deja registros parciales.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar la validación real del proyecto y del período, sin reemplazarla con
mocks o métodos exclusivos de test.
