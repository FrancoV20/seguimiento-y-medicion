# Quickstart: Interfaz web de Sprints

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

Desde la raíz del repositorio, iniciar PostgreSQL y aplicar migraciones:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo con historias en el Backlog:

1. Si no hay integrantes, cargarlos con la semilla de HU-21 (`db/seeds/members.sql`).
2. Crear un proyecto en `http://localhost:8080/proyectos` (HU-21).
3. Elegirlo como proyecto activo en `http://localhost:8080/contexto/proyecto` (HU-20b).
4. Cargar al menos tres historias en `http://localhost:8080/backlog` (HU-22).

Iniciar el servidor y abrir `http://localhost:8080/sprints`:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Si HU-21 o HU-22 todavía no están integradas, las pruebas automáticas no las necesitan: crean sus
propios proyectos, historias y Sprints. Para la prueba manual se pueden insertar el proyecto y las
historias con SQL. Esta feature no agrega migraciones; si alguna vez hiciera falta una, toma el último
número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia.

Si en tu base de desarrollo quedó un Sprint activo de prueba y querés crear otro, cerralo a mano
(HU-15 todavía no existe) o recreá la base con `docker compose down -v`:

```sql
UPDATE sprints SET status = 'Finalizado' WHERE project_id = <ID_DEL_PROYECTO> AND status = 'Activo';
```

## Manual check: recorrido completo con un proyecto activo

Con el servidor en ejecución, un proyecto activo elegido y al menos tres historias "Pendiente" en el
Backlog (ver "Run the application"):

1. Abrir `http://localhost:8080/sprints`: se ve "No hay un Sprint activo" y el formulario de creación.
2. Enviar el formulario con el Goal vacío o con la fecha de fin anterior a la de inicio: el mensaje
   aparece junto al campo y los datos y las marcas se conservan.
3. Completar Goal, fechas y marcar dos historias; "Crear Sprint": se ve "Sprint creado correctamente"
   una sola vez, el Sprint (por ejemplo "Sprint 1") con sus fechas `dd/mm/aaaa` y las dos historias
   "En Sprint". Recargar la página no duplica el Sprint ni repite el mensaje.
4. En "Agregar historias", marcar la tercera historia y enviar: "Historias agregadas correctamente".
   Enviar sin marcar ninguna: "Elija al menos una historia".
5. Pulsar "Quitar" en una historia "En Sprint" sin horas: "Historia quitada del Sprint" y la historia
   vuelve a estar disponible en la lista.
6. Pasar una historia del Sprint a "En progreso" y registrar horas en otra (por SQL si HU-14 o HU-24
   no están integradas):

   ```sql
   UPDATE user_stories SET status = 'En progreso' WHERE id = <ID_HISTORIA_1>;
   INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
   VALUES (<ID_HISTORIA_2>, <ID_INTEGRANTE>, CURRENT_DATE, 'Prueba manual', 1);
   ```

   Al recargar, "Quitar" queda deshabilitado con "Ya está en progreso" y "Tiene horas registradas".
7. Con un Sprint activo, la pantalla no ofrece el formulario de creación e informa "Ya hay un Sprint
   activo (Sprint N)".
8. Cerrar a mano el Sprint de prueba (HU-15 todavía no existe) y recargar: vuelve a ofrecerse el
   formulario de creación.

   ```sql
   UPDATE sprints SET status = 'Finalizado' WHERE project_id = <ID_DEL_PROYECTO> AND status = 'Activo';
   ```

   Las historias que quedaron asignadas siguen con su asignación; para empezar de cero, recrear la
   base con `docker compose down -v` y volver a migrar.

### Nota de puerto

`docker-compose.yml` publica PostgreSQL en el puerto `5433` del host hacia el `5432` del contenedor.
Si tu `docker-compose.override.yml` (no versionado) publica otro puerto, usá ese. `DATABASE_URL`
siempre lleva el puerto del host:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
```

Las credenciales personales van solo en tu `docker-compose.override.yml` y en tu variable de entorno,
nunca en archivos del repositorio. El servidor web escucha en `:8080` salvo que se defina `ADDR`.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Crear un Sprint:
   - con Goal, fechas y tres historias pendientes: identificador "Sprint 1" no editable, historias
     "En Sprint" y mensaje de éxito una sola vez;
   - con dos proyectos, el Sprint y las historias quedan solo en el activo;
   - sin historias: se crea con la advertencia;
   - recargar la página después de crear: no se duplica ni se repite el mensaje.
2. Un solo Sprint activo (la regla es de FIX-HU-04):
   - con un Sprint activo no se ofrece el formulario de creación y se informa "Ya hay un Sprint
     activo (Sprint N)";
   - un `POST /sprints` igualmente recibido responde 409 con "Ya hay un Sprint activo (Sprint N)" y no
     crea nada ni consume un número de Sprint.
3. Lista de historias:
   - disponibles marcables; asignadas a un Sprint y no pendientes, deshabilitadas con el motivo;
   - un identificador de historia repetido se toma una vez; no numérico o de otro proyecto se
     rechaza con un mensaje junto a la lista.
4. Errores de validación:
   - Goal vacío, fecha de fin anterior al inicio y fecha de fin igual a la de inicio: mensajes de
     HU-04 junto a cada campo, datos y marcas conservados, sin crear y sin consumir número;
   - fecha faltante ("Falta la fecha de inicio" o "Falta la fecha de fin") o de formato inválido
     ("La fecha no tiene un formato válido"): mensaje junto al campo.
5. Conflictos y fallas:
   - una historia que dejó de estar disponible: no se crea el Sprint y se informa cuál;
   - falla de guardado: "La operación no pudo completarse", datos conservados, sin éxito y sin Sprint
     ni asignaciones parciales.
6. Contexto:
   - sin proyecto activo, guardar redirige a la selección de proyecto con el mensaje de HU-20b y no
     guarda nada;
   - un identificador de proyecto o de Sprint enviado en el formulario se ignora.
7. Ver el Sprint activo:
   - identificador, Goal, fechas `dd/mm/aaaa` e historias con su estado;
   - sin Sprint activo: "No hay un Sprint activo" y el formulario;
   - falla de lectura: página de error de la aplicación, no una pantalla incompleta.
8. Agregar historias:
   - las marcadas disponibles quedan "En Sprint", aparecen en el Sprint y se muestra "Historias
     agregadas correctamente";
   - sin marcar ninguna: "Elija al menos una historia" y sin cambios;
   - una historia no disponible: no se asigna ninguna y se informa cuál.
9. Quitar historias:
   - una historia "En Sprint" sin horas vuelve a "Pendiente", sale del Sprint, se muestra "Historia
     quitada del Sprint" y vuelve a estar disponible;
   - en progreso, hecha o con horas: "Quitar" deshabilitado con el motivo;
   - la regla se valida en el dominio: un envío directo sobre una historia en progreso, hecha o con
     horas se rechaza con el motivo y sin cambios; una historia de otro proyecto o Sprint se rechaza.
10. Seguridad y detalles internos:
    - un Goal con `<script>` se ve como texto;
    - ninguna respuesta de error contiene texto de consultas ni rutas.
11. PostgreSQL:
    - `PostgreSQLSprintCatalog` devuelve el Sprint activo del proyecto correcto, sus historias con el
      indicador de horas, y las opciones con su motivo, contra PostgreSQL real y con fixtures propias.

## Manual check: usabilidad (SC-009 y SC-010)

En la Sprint Review, con 5 personas que no hayan visto la pantalla y un Backlog con al menos tres
historias pendientes:

1. Pedirles que creen un Sprint con Goal, fechas a elección y tres historias, sin ayuda.
2. Cronometrar desde que abren "Sprints" hasta que ven el mensaje de éxito.
3. Anotar si lo lograron en el primer intento (sin ver un mensaje de error).
4. Guardar los resultados en `specs/023-web-sprints/evidencia-usabilidad.txt` con una línea por
   persona: tiempo y si fue al primer intento, más el promedio y el porcentaje de éxito.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las
pruebas deben ejecutar las reglas reales de HU-04 y el almacenamiento real, no reemplazarlos con
mocks; solo se usa un almacenamiento que falla para simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/023-web-sprints/`. En PowerShell, ejecutar `chcp 65001` antes para que
las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/023-web-sprints/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/023-web-sprints/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/023-web-sprints/cobertura.txt
```

## Manual breve de usuario: pantalla "Sprints"

La pantalla "Sprints" (`/sprints`) planifica el Sprint del proyecto activo. Si no hay un proyecto
elegido, la aplicación pide elegirlo primero.

**Crear un Sprint** (solo cuando no hay uno activo):

1. Escribir el Sprint Goal y elegir las fechas de inicio y de fin. La fecha de fin debe ser posterior
   a la de inicio.
2. Marcar las historias del Backlog que entran al Sprint. Las que no se pueden elegir aparecen
   deshabilitadas con el motivo ("Ya está asignada al Sprint N" o "No está pendiente"). También se
   puede crear sin historias y agregarlas después.
3. Pulsar "Crear Sprint". El número del Sprint lo asigna la aplicación y no se puede cambiar.

**Ver el Sprint activo**: se muestran su número, el Sprint Goal, las fechas y sus historias con los
Story Points y el estado. Solo puede haber un Sprint activo por proyecto.

**Agregar historias**: en "Agregar historias", marcar una o más historias disponibles y pulsar
"Agregar historias". Pasan al estado "En Sprint".

**Quitar una historia**: pulsar "Quitar" en su fila. La historia vuelve al Backlog como "Pendiente".
Solo se pueden quitar historias "En Sprint" sin horas registradas; si no, el botón aparece
deshabilitado con el motivo: "Ya está en progreso", "Ya está hecha" o "Tiene horas registradas".

**Mensajes**: los errores de un campo aparecen junto a ese campo y se conservan los datos cargados.
Si aparece "La operación no pudo completarse", no se guardó nada: volver a intentarlo más tarde.
