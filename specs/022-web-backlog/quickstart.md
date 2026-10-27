# Quickstart: Interfaz web del Product Backlog

## Requisitos

- Go 1.27.1 o compatible con la versión declarada en `go.mod`.
- Docker Desktop y Docker Compose disponibles.
- PostgreSQL 16 del servicio `postgres` de `docker-compose.yml`.
- Herramienta `golang-migrate/migrate` instalada para aplicar migraciones.

La aplicación requiere `DATABASE_URL`. Compose publica PostgreSQL en el puerto local `5433` (puerto
interno del contenedor: `5432`). El compose principal define los valores de desarrollo `sym_user` /
`sym_pass` para la base `seguimiento_y_medicion`:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@127.0.0.1:5433/seguimiento_y_medicion?sslmode=disable"
```

Si el entorno local tiene credenciales distintas, ajusta `DATABASE_URL` solo en tu entorno. No
documentes ni agregues credenciales personales o de producción a archivos versionados.

## Iniciar la aplicación

Desde la raíz del repositorio, inicia PostgreSQL y aplica las migraciones:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Verifica el estado del servicio y luego inicia el servidor:

```powershell
docker compose ps postgres
go run src/cmd/server/main.go
```

La dirección predeterminada es `:8080`. Si el puerto está ocupado, configura `ADDR` antes de iniciar;
por ejemplo, para usar el puerto 8081:

```powershell
$env:ADDR = ":8081"
go run src/cmd/server/main.go
```

Abre `http://localhost:8081/contexto/proyecto` y selecciona el proyecto; luego visita
`http://localhost:8081/backlog`. Si usas la dirección predeterminada, reemplaza `8081` por `8080`.

### Contexto de proyecto requerido

Debe existir al menos un proyecto activo en la base de datos para superar la pantalla de selección
inicial. Selecciónalo en `/contexto/proyecto`; HU-20b lo conserva en el contexto de la sesión. Si la
lista de proyectos está vacía, crea o carga uno mediante la funcionalidad de proyectos disponible en
el entorno (HU-21) y vuelve a la pantalla de selección. La pantalla Backlog siempre toma el proyecto
activo del contexto del servidor, no de un campo enviado por el formulario.

HU-22 no incorpora migraciones. Las pruebas de integración crean sus propios proyectos e historias,
por lo que no requieren datos de negocio preexistentes.

## Ejecutar las pruebas

Con PostgreSQL iniciado y `DATABASE_URL` configurada en la misma sesión de PowerShell:

```powershell
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración compartan la base
simultáneamente.

## Escenarios cubiertos

1. **Alta válida**: alta con título, descripción, prioridad, criterios, Story Points y horas; mensaje
   de éxito; historia visible como "Pendiente"; aislamiento por proyecto y recarga sin duplicar.
2. **Criterios dinámicos**: agregar y quitar campos sin persistir, conservar lo escrito, no quitar el
   último campo e ignorar criterios en blanco al guardar. Enter ejecuta la acción `guardar`.
3. **Estimaciones opcionales**: Story Points ausentes generan advertencia y "Sin estimar"; horas
   ausentes no generan advertencia; coma y punto decimal se aceptan y se muestran con coma.
   `9999,99` es válido.
4. **Validación**: criterios sin texto, título o descripción vacíos, prioridad manipulada, Story
   Points fuera de Fibonacci y horas inválidas (`0`, negativas, no numéricas, con más de dos
   decimales o superiores a `9999,99`). Se conserva el formulario y no se persiste la historia.
5. **Errores y contexto**: la falla de guardado responde con el mensaje genérico, sin detalles
   internos; la falta de proyecto activo redirige a la selección sin guardar.
6. **Listado**: identificador, título, prioridad, Story Points, horas y estado; aislamiento por
   proyecto; estimaciones nulas como "Sin estimar"; estado vacío "Aún no hay historias"; error de
   lectura sin renderizar una lista parcial.
7. **PostgreSQL y seguridad**: el catálogo integra con la base y usa fixtures propios; la plantilla
   escapa contenido dinámico y no expone consultas ni rutas internas en errores.

## Evidencia de validación

Para registrar la ejecución final de las pruebas y la cobertura en UTF-8:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/022-web-backlog/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/022-web-backlog/cobertura.txt
```
