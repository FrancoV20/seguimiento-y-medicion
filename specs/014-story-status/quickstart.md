# Quickstart: Actualización de estado de historias (Tablero)

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml` o
  `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo con un Sprint activo y historias asignadas:

1. Crear el proyecto (HU-21) y elegirlo en `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear las historias (HU-22) y un Sprint con tres historias (HU-23). Si esas historias todavía no
   están integradas, insertar los datos por SQL respetando el esquema de `data-model.md` y dejar las
   historias en `'En Sprint'`.
3. Abrir `http://localhost:8080/tablero`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Las pruebas automáticas no dependen de HU-21, HU-22 ni HU-23: crean sus propios proyectos, historias,
Sprints y horas. Esta feature no agrega migraciones.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Regla de transiciones (unitaria): las tres permitidas, y rechazadas En Sprint → Hecho, En progreso →
   En Sprint, Hecho → En Sprint, cualquier destino Pendiente y las repetidas; `NextAction` y `ActionTransition`
   de cada estado.
2. Servicio y almacenamiento (PostgreSQL real):
   - comenzar, completar y reabrir actualizan el estado;
   - completar sin horas devuelve la advertencia; con horas, no;
   - historia sin Sprint, de un Sprint finalizado, de otro proyecto o inexistente: rechazada y sin
     cambios;
   - estado ya aplicado: no cambia y no es un error de almacenamiento;
   - estado esperado distinto del real: `ErrStatusChanged` y sin cambios;
   - acción que no corresponde al estado real (`comenzar` sobre Hecho, `reabrir` sobre En Sprint):
     `ErrStatusChanged` y sin cambios;
   - dos cambios simultáneos sobre la misma historia: solo uno se aplica.
3. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - cada acción válida: 303 a `/tablero`, mensaje una sola vez y estado nuevo; con dos proyectos, solo
     afecta al activo;
   - Story Points completados antes y después de completar, reabrir y enviar dos veces;
   - cada rechazo con su mensaje y código (409 o 422) y el estado sin cambios;
   - falla de guardado: "La operación no pudo completarse", sin éxito y sin cambios;
   - recarga después de un cambio: no se repite ni repite el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada cambia; un campo de proyecto enviado se ignora.
4. Vista:
   - Sprint, grupos por estado, Story Points completados de planificados y una sola acción por historia;
   - sin Sprint activo y Sprint sin historias: sus mensajes y ninguna acción;
   - historia sin Story Points: "Sin estimar"; título con `<script>` se ve como texto;
   - falla de lectura: página de error de la aplicación.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.
6. PostgreSQL: `PostgreSQLStoryStatusStore` y `PostgreSQLBoardCatalog` contra PostgreSQL real con
   fixtures propias.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas
ejecutan las reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para
simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/014-story-status/`. En PowerShell, ejecutar `chcp 65001` antes para que
las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/cobertura.txt
```