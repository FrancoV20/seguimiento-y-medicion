# Quickstart: Cierre de Sprint

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-14, HU-07, HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`; en tu rama,
  traerlas con `git merge origin/main`.
- Variable `DATABASE_URL` configurada. `docker-compose.yml` publica el puerto del host `5433` hacia el
  puerto `5432` del contenedor; la URL local usa `localhost:5433`. Ajustar si
  `docker-compose.override.yml` cambia el mapeo:
  `postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

La migración de esta feature se aplica con el resto. Probar `up` y `down`:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
migrate -path db/migrations -database $env:DATABASE_URL down 1
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo con un Sprint activo y historias "Hecho", "En progreso" y
"En Sprint":

1. Crear el proyecto (HU-21) y elegirlo en `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear las historias (HU-22), un Sprint con ellas (HU-23) y pasarlas de estado en `/tablero` (HU-14).
   Si esas historias todavía no están integradas, insertar los datos por SQL respetando el esquema de
   `data-model.md`.
3. Abrir `http://localhost:8080/cierre`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

### Prueba manual del cierre

El puerto del host para PostgreSQL es `5433`, mapeado al puerto `5432` del contenedor. Configurar
`DATABASE_URL` con `localhost:5433` antes de ejecutar migraciones o iniciar el servidor. Sustituir las
credenciales locales de la URL por las propias y no subirlas al repositorio.

1. Iniciar PostgreSQL con `docker compose up -d postgres` y aplicar las migraciones usando
   `DATABASE_URL`.
2. Crear y seleccionar un proyecto por HU-20b. Crear un Sprint activo y asignarle al menos tres
   historias: una en "Hecho", una en "En progreso" y una en "En Sprint". Usar Story Points válidos para
   HU-07. Si HU-21, HU-22 o HU-23 no están disponibles, preparar esos datos por SQL respetando
   `data-model.md`.
3. Ejecutar `go run ./src/cmd/server` y abrir `http://localhost:8080/cierre`.
4. Verificar que la vista previa separa la historia completada de las que volverán al Backlog. Confirmar
   el cierre: debe redirigir con el mensaje de resultado, conservar "Hecho" en el Sprint finalizado y
   devolver las otras historias al Backlog en estado "Pendiente".

Las pruebas automáticas no dependen de HU-21, HU-22, HU-23 ni de las pantallas de HU-14: crean sus propios
proyectos, historias, Sprints y horas.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al compartir
una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Migración: `up` y `down` en ambos sentidos; la tabla tiene la clave `(sprint_id, story_id)` y borra en
   cascada.
2. Servicio y almacenamiento (PostgreSQL real):
   - cierre válido: Sprint "Finalizado"; las "En Sprint" y "En progreso" pasan a "Pendiente" sin
     asignación ni `active_sprint_id`; las "Hecho" conservan estado y asignación;
   - registro del Sprint: una fila por historia con estado final y Story Points; no cambia si una
     historia devuelta se asigna a otro Sprint o se edita;
   - Sprint sin historias y Sprint sin "Hecho": se cierran; HU-07 calcula velocidad cero;
   - Sprint ya cerrado, inexistente o de otro proyecto: rechazado y sin cambios;
   - datos inválidos según HU-07: `ErrCloseBlocked` con las historias a corregir y sin cambios;
   - falla a mitad del cierre (por ejemplo en el último paso): se revierte todo;
   - dos cierres simultáneos: solo uno se aplica;
   - las historias devueltas se pueden asignar a un Sprint nuevo (HU-04) y no hay otro Sprint activo hasta
     que se crea;
   - registrar horas (HU-03) en una historia del Sprint finalizado o devuelta: rechazado.
3. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - cierre válido: 303 a `/cierre`, mensaje con número y cantidad una sola vez; con dos proyectos, solo
     cierra el activo;
   - cada rechazo con su mensaje y código (409 o 422) y sin cambios; un `sprint_id` de un Sprint ya
     cerrado no cierra el Sprint nuevo;
   - falla de guardado: "La operación no pudo completarse", Sprint activo y sin cambios parciales;
   - recarga después de cerrar: no se repite ni repite el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada cambia.
4. Vista:
   - resumen con Sprint, completadas, devueltas y Story Points, y botón; título con `<script>` visto como
     texto;
   - sin Sprint activo: "No hay un Sprint activo para cerrar" y sin botón;
   - falla de lectura: página de error de la aplicación.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas
ejecutan las reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para
simular la falla de guardado y de lectura, y los datos inválidos de HU-07 se arman como en las pruebas de
HU-07.

Guardar la evidencia en `specs/015-sprint-close/`. En PowerShell, ejecutar `chcp 65001` antes para que las
tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/015-sprint-close/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/015-sprint-close/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/015-sprint-close/cobertura.txt
```