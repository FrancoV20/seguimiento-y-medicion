# Quickstart: Interfaz web de proyectos

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-01, FIX-HU-01, HU-20a y HU-20b (US1 y US2) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main` y aplicar las migraciones nuevas (la de FIX-HU-01 cambia la restricción de
  fechas de `projects`).
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml`
  o `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

Desde la raíz del repositorio, iniciar PostgreSQL y aplicar migraciones:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Cargar los integrantes de prueba (una vez por base local; se puede repetir sin duplicar). Ajustar
`USUARIO` al de tu base:

```powershell
Get-Content -Encoding UTF8 db/seeds/members.sql | docker exec -i seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion
```

Iniciar el servidor y abrir `http://localhost:8080/proyectos`:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Esta feature no agrega migraciones (la de FIX-HU-01 ya viene en `main`). La semilla no es una
migración y no consume número; si alguna vez hiciera falta una migración, toma el último número de
`main` más uno, sin importar el número de la historia.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base. Las pruebas crean sus propios integrantes y proyectos; no necesitan la semilla.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Alta válida:
   - proyecto con nombre, integrantes y fechas: mensaje de éxito y aparece en la lista;
   - misma fecha de inicio y de fin: se rechaza (FIX-HU-01) con el mensaje de fechas junto a la
     fecha de fin, datos conservados y sin crear;
   - fecha de fin un día posterior a la de inicio: se crea;
   - recargar la página después de crear: no se duplica ni se repite el mensaje;
   - crear no cambia el proyecto activo.
2. Integrantes disponibles:
   - el selector ofrece solo integrantes existentes;
   - sin integrantes cargados: "No hay integrantes disponibles" y no se puede crear;
   - identificadores repetidos se toman una vez.
3. Errores de validación:
   - nombre vacío o solo espacios, fecha de fin anterior a la de inicio y sin integrantes: mensajes de
     HU-01, cada uno junto a su campo, datos conservados, sin crear;
   - fecha de inicio vacía: "Falta la fecha de inicio"; fecha de fin vacía: "Falta la fecha de fin";
   - fecha con formato inválido: "La fecha no tiene un formato válido";
   - integrante no numérico o inexistente: "El integrante seleccionado no es válido".
4. Falla de guardado:
   - "La operación no pudo completarse", datos conservados, sin mensaje de éxito y sin el
     proyecto en la lista.
5. Lista:
   - nombre, fechas y cantidad de integrantes, con los más recientes primero;
   - sin proyectos: "Aún no hay proyectos" y el formulario;
   - falla de lectura: página de error de la aplicación, no un formulario incompleto.
6. Seguridad y detalles internos:
   - un nombre con `<script>` se ve como texto;
   - ninguna respuesta de error contiene texto de consultas ni rutas.
7. PostgreSQL:
   - `PostgreSQLProjectCatalog` lista proyectos con su cantidad de integrantes y el catálogo de
     integrantes contra PostgreSQL real, con fixtures propias.

## Manual check: usabilidad (SC-003 y SC-004)

En la Sprint Review, con 5 personas que no hayan visto la pantalla:

1. Pedirles que creen un proyecto con 2 integrantes y fechas a elección, sin ayuda.
2. Cronometrar desde que abren "Gestión de Proyectos" hasta que ven el mensaje de éxito.
3. Anotar si lo lograron en el primer intento (sin ver un mensaje de error).
4. Guardar los resultados en `specs/021-web-projects/evidencia-usabilidad.txt` con una línea por
   persona: tiempo y si fue al primer intento, más el promedio y el porcentaje de éxito.

## Manual breve de usuario

Agregar al manual breve de usuario del repositorio (en la ruta que acordó el equipo) la sección
"Gestión de Proyectos": cómo crear un proyecto, qué significa cada mensaje de error y que la fecha
de fin debe ser posterior a la de inicio. Incluir una captura de la pantalla.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar las reglas reales de HU-01 y el almacenamiento real, no reemplazarlos con
mocks; solo se usa un almacenamiento que falla para simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/021-web-projects/`. En PowerShell, ejecutar `chcp 65001` antes para
que las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/021-web-projects/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/021-web-projects/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/021-web-projects/cobertura.txt
```
