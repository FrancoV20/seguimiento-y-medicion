# Quickstart: Formularios seguros y contexto de trabajo

## Prerequisites

- HU-20a integrada: servidor, layout, `Section`, `Server.Register`, `Server.Use`, `ActiveContext`,
  `Fail` y `RenderError` disponibles.
- Go 1.27.1 y Docker Desktop instalados y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- Migración `001` (proyectos e integrantes, HU-01) aplicada. Esta feature no agrega migraciones.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml`
  o `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.

## Run the tests

Desde la raíz del repositorio:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base. Si la base local ya aplicó una migración de número mayor y falta una de
número menor, recrearla con `docker compose down -v` y volver a migrar.

## Datos de prueba

Las pruebas de integración y BDD crean sus propios proyectos e integrantes (fixtures) y los eliminan al terminar. Para probar a mano, hasta que exista la pantalla de alta (HU-21), cargá datos con SQL. Reemplazá `USUARIO` por el de tu `DATABASE_URL`:

```powershell
@"
INSERT INTO projects (name, start_date, end_date) VALUES
  ('Demo HU20b - Proyecto Alfa', '2026-09-01', '2026-12-15'),
  ('Demo HU20b - Proyecto Beta', '2026-10-01', '2027-02-28');
INSERT INTO members (display_name) VALUES ('Demo Ana'), ('Demo Beto'), ('Demo Carla');
INSERT INTO project_members (project_id, member_id)
SELECT p.id, m.id FROM projects p, members m
WHERE (p.name = 'Demo HU20b - Proyecto Alfa' AND m.display_name IN ('Demo Ana', 'Demo Beto'))
   OR (p.name = 'Demo HU20b - Proyecto Beta' AND m.display_name = 'Demo Carla');
"@ | docker exec -i seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion
```

Para borrarlos:

```powershell
"DELETE FROM projects WHERE name LIKE 'Demo HU20b - %'; DELETE FROM members WHERE display_name LIKE 'Demo %';" | docker exec -i seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion
```

Esquema que se lee (`001_create_projects.up.sql`): `projects(id, name, start_date, end_date, status)`, `members(id, display_name)` y `project_members(project_id, member_id)`.

Atención: la prueba de integración de HU-01 hace `TRUNCATE ... CASCADE` y borra los datos de la base local cada vez que corrés `go test ./...` con `DATABASE_URL` definida. Probá a mano antes de correr la suite completa, o volvé a cargar los datos después.

Las secciones reales (Backlog, Sprints, Esfuerzo, Métricas) llegan en HU-22 a HU-25. Mientras tanto, las pruebas registran secciones de prueba: una sin requisitos, una con `RequiresProject` y una con `RequiresMember`.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Mensajes y formularios:
   - formato por tipo de mensaje, con etiqueta de texto además del color;
   - errores junto al campo con datos conservados;
   - texto único "La operación no pudo completarse", sin mensaje de éxito;
   - envío válido: 303 y mensaje una sola vez; recarga sin duplicar.
2. Seguridad:
   - texto con `<script>` mostrado como texto;
   - envío con `Sec-Fetch-Site: cross-site` o `Origin` ajeno rechazado con 403 y mensaje claro;
     mismo origen aceptado;
   - (extra, opcional) cuerpo mayor a 1 MiB rechazado con 413 y cabeceras de seguridad.
3. Selección de proyecto:
   - elegir un proyecto y ver solo sus datos con dos proyectos de datos distintos;
   - mantener la selección al navegar entre secciones y ver el nombre en la estructura común;
   - sin proyectos: "Aún no hay proyectos" y acceso a crear el primero solo si la sección
     Proyectos está registrada;
   - con un solo proyecto: no se autoselecciona;
   - cambiar de proyecto limpia el integrante; volver a elegir el mismo lo conserva;
   - cerrar el navegador (cookie de sesión nueva) vuelve a pedir el proyecto.
4. Datos inválidos:
   - identificador inexistente, no numérico, negativo o muy grande: mensaje claro, selección
     anterior conservada, sin error interno;
   - cookie manipulada: tratada como ausencia y borrada;
   - proyecto eliminado después de seleccionado: selección limpiada con advertencia.
5. Identificación del integrante:
   - el selector muestra solo integrantes del proyecto activo;
   - la identificación se mantiene y se muestra en el layout;
   - acción que requiere integrante sin identificarse: se pide identificarse y se vuelve;
   - integrante de otro proyecto: acción rechazada y identificación limpiada;
   - integrante quitado del proyecto: identificación limpiada con advertencia.
6. Secciones protegidas:
   - sin proyecto, las secciones con `RequiresProject` redirigen a la selección con mensaje;
   - al elegir, vuelve a la sección pedida;
   - destinos `volver_a` externos o malformados (`//otro.com`, `https://...`, `\\`) van a `/`;
   - secciones sin requisitos accesibles sin contexto.
7. Disponibilidad:
   - con la base de datos caída no se limpia la selección y se muestra el 503 de HU-20a.
8. PostgreSQL:
   - `PostgreSQLContextStore` lista proyectos e integrantes y resuelve el contexto contra
     PostgreSQL real;
   - `ResolveContext` distingue "no existe" de "no disponible".

## Manual check

1. Levantar la base y arrancar el servidor en un puerto libre (el 8080 puede estar ocupado):

```powershell
   docker compose up -d postgres
   $env:ADDR = ":8081"
   go run ./src/cmd/server
```

2. Abrir `http://localhost:8081/contexto/proyecto` y comprobar la lista de proyectos. Elegir uno: su nombre aparece en el encabezado.
3. Apretar "Identificarme": solo aparecen los integrantes de ese proyecto. Elegir uno y comprobar "Integrante: ..." en el encabezado. Cambiar de proyecto: la identificación desaparece.
4. En las herramientas del navegador (F12, Application, Cookies): `ctx_proyecto` y `ctx_integrante` con HttpOnly, SameSite Lax y Expires "Session".
5. Abrir una ventana privada, elegir proyecto, cerrar todas las ventanas privadas y abrir otra: se pide elegir de nuevo. (Un navegador con "continuar donde lo dejaste" conserva las cookies de sesión.)
6. Comprobar el rechazo de envíos de otro origen. Debe imprimir `403`:

```powershell
   try { Invoke-WebRequest -Uri http://localhost:8081/contexto/proyecto -Method Post -Headers @{"Sec-Fetch-Site"="cross-site"} -Body "proyecto_id=1" -UseBasicParsing } catch { $_.Exception.Response.StatusCode.value__ }
```

   Este paso también confirma el cableado de `main.go`, que las pruebas automáticas no cubren.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar el servidor, el middleware y el almacenamiento reales, no reemplazarlos con
mocks o métodos exclusivos de test.
