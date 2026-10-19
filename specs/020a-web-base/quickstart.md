# Quickstart: Base de la aplicación web

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- Variable `DATABASE_URL` configurada. El `docker-compose.yml` publica PostgreSQL en el puerto
  `5433`; si tu `docker-compose.override.yml` usa otro puerto, usuario o clave, ajustá la URL:
  `postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable`.
- En PowerShell la variable se pierde al cerrar la terminal; para dejarla fija:
  `[Environment]::SetEnvironmentVariable("DATABASE_URL", "<url>", "User")`.
- Opcional: `ADDR` para cambiar el puerto del servidor (por defecto `:8080`).
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

Desde la raíz del repositorio, iniciar PostgreSQL y aplicar migraciones:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database "$env:DATABASE_URL" up
```

Iniciar el servidor y abrir `http://localhost:8080`:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Esta feature no agrega migraciones; solo exige que las existentes estén aplicadas. Si alguna vez
hiciera falta una, toma el último número de `main` más uno (hoy la próxima es la 008), sin
importar el número de la historia. Si el servidor informa que el esquema está desactualizado, o
si una migración nueva no se aplicó por tener un número menor al de la base local, recrear la
base con `docker compose down -v`, volver a levantarla y migrar.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Arranque:
   - página de inicio con menú;
   - falta `DATABASE_URL`: mensaje con el nombre de la variable y sin valores;
   - base de datos lenta: reintenta y arranca; base de datos caída: falla sin mostrar la clave;
   - migraciones pendientes y estado `dirty`: no arranca y explica qué hacer;
   - puerto en uso o no válido: informa el puerto afectado y no arranca;
   - detención ordenada con una solicitud en curso.
2. Estructura común:
   - menú con las secciones registradas y la actual resaltada;
   - sección no registrada ausente; dirección duplicada: el arranque falla;
   - "Sin proyecto seleccionado" sin contexto;
   - ninguna referencia a recursos externos en el HTML;
   - tildes, ñ y símbolos correctos.
3. Errores:
   - dirección inexistente y parámetro malformado, negativo o muy grande: 404 con estructura
     común y enlace al inicio;
   - método no soportado: 405 con estructura común y servidor activo;
   - pánico en un handler: página 500 genérica, servidor activo y detalle solo en el log;
   - falla de plantilla a mitad del render: página de error completa, nunca HTML parcial;
   - base de datos detenida con el servidor activo: 503; al volver, respuesta normal sin
     reiniciar;
   - ninguna respuesta de error contiene rutas, SQL, trazas ni `DATABASE_URL`.
4. PostgreSQL:
   - `PostgreSQLSchemaVersionReader` lee versión y `dirty` de `schema_migrations` real;
   - todas las migraciones `.up.sql` se aplican en orden sobre una base descartable vacía (la
     prueba la crea y la borra; nunca usa la base de desarrollo).

## Manual check: caída de la base de datos

Con el servidor en ejecución:

```powershell
docker compose stop postgres
# abrir cualquier página que use la base: debe verse "El servicio no está disponible temporalmente"
docker compose start postgres
# recargar: debe funcionar sin reiniciar el servidor
```

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar el servidor y las reglas reales, no reemplazarlas con mocks o métodos
exclusivos de test.
