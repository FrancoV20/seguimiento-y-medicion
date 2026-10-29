# Quickstart: Horas estimadas por historia (Estimaciones)

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-02, HU-03, HU-22, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada con el puerto **del host** (ver "Nota de puerto"):
  `postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

### Nota de puerto

`docker-compose.yml` define el servicio `postgres` (imagen `postgres:16-alpine`, contenedor
`seguimiento-y-medicion-db`, base `seguimiento_y_medicion`, usuario `sym_user`) con chequeo de salud
`pg_isready` y publica el puerto `5433` del host hacia el `5432` del contenedor. Si tu
`docker-compose.override.yml` (no versionado) publica otro puerto o cambia las credenciales, usá esos
valores. En PowerShell:

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
```

El CI de GitHub Actions usa su propio PostgreSQL en el puerto `5432`; los ejemplos con `5432` de otros
`quickstart.md` corresponden a ese caso. El servidor web escucha en `:8080` salvo que se defina `ADDR`.

## Run the application

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo con historias, algunas con horas reales y una de un
Sprint finalizado:

1. Crear el proyecto (HU-21) y elegirlo en `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear al menos tres historias en `http://localhost:8080/backlog` (HU-22), una con horas estimadas y
   otra sin.
3. Cargar horas reales a una historia (HU-24, o por SQL en `effort_logs` si no está integrada).
4. Para probar el bloqueo, asignar una historia a un Sprint y dejarlo finalizado a mano:

```sql
UPDATE sprints SET status = 'Finalizado' WHERE project_id = <ID_DEL_PROYECTO> AND status = 'Activo';
```

5. Abrir `http://localhost:8080/estimaciones`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Las pruebas automáticas no dependen de HU-21, HU-22 ni HU-23: crean sus propios proyectos, historias,
Sprints y horas. Esta feature no agrega migraciones.

## Manual check: recorrido completo

Con el servidor en ejecución y un proyecto activo con al menos cuatro historias (ver "Run the
application"): una sin horas estimadas ni Story Points, una con horas estimadas, una con horas reales
registradas y una asignada a un Sprint finalizado.

1. Abrir `http://localhost:8080/estimaciones`. Cada fila muestra título, estado, Story Points, horas
   estimadas y horas reales, las historias más recientes primero. La historia sin estimar muestra "Sin
   estimar" (nunca 0) y una historia sin horas reales muestra 0.
2. En la historia sin estimar, escribir `8` y pulsar "Guardar": se ve "Horas estimadas registradas" una
   sola vez y la fila muestra 8. Recargar la página (F5) no repite el mensaje ni el guardado.
3. En la historia con horas reales, escribir `1,5` (o `1.5`) y "Guardar": "Horas estimadas actualizadas",
   la fila muestra 1,5 y las horas reales no cambian.
4. Probar valores rechazados en cualquier fila editable: `0`, `-1`, `abc`, vacío y `1,555` muestran "El
   valor de horas no es válido" junto al campo; `10000` muestra "Las horas estimadas no pueden superar
   9999,99". En todos los casos lo escrito queda en el campo y el valor anterior no cambia.
5. La historia del Sprint finalizado tiene el campo y el botón deshabilitados con el motivo "Sprint
   finalizado". Para probar el rechazo directo, quitar el atributo `disabled` con las herramientas del
   navegador (F12) y enviar: responde 409 con "La historia pertenece a un Sprint finalizado; sus horas
   estimadas no se pueden modificar" y el valor no cambia.
6. Elegir un proyecto sin historias: se ve "Aún no hay historias".
7. Con la página abierta, detener PostgreSQL (`docker compose stop postgres`) y recargar: se ve la página
   de error de la aplicación, sin texto de consultas ni rutas. Volver a iniciarlo con
   `docker compose start postgres`.

Datos de apoyo por SQL (reemplazar los identificadores):

```sql
-- horas reales para una historia
INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
VALUES (<ID_HISTORIA>, <ID_INTEGRANTE>, CURRENT_DATE, 'Prueba manual', 2.5);

-- historia en un Sprint finalizado (HU-15 todavía no cierra Sprints)
INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
VALUES (<ID_PROYECTO>, 99, 'Sprint de prueba HU-17', DATE '2026-10-01', DATE '2026-10-15', 'Finalizado')
RETURNING id;
INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES (<ID_SPRINT>, <ID_HISTORIA>);
```

### Nota de puerto (recordatorio)

`DATABASE_URL` usa el puerto del host: `5433` con el `docker-compose.yml` del repositorio, o el que
publique tu `docker-compose.override.yml`. Si las pruebas de integración aparecen como `SKIP` o fallan
con `connection refused`, verificar que Docker Desktop esté en ejecución y que `docker compose ps`
muestre el contenedor `healthy`. Después de traer `main`, ejecutar
`migrate -path db/migrations -database $env:DATABASE_URL up`.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Servicio y almacenamiento (PostgreSQL real):
   - registrar horas en una historia sin estimación y modificar una existente (el valor reemplaza al
     anterior);
   - solo cambia `estimated_hours`: las horas reales, los Story Points y los demás campos no cambian;
   - valores válidos exactos (`8`, `1,5`, `1.5`, `0,01`, `9999,99`) y rechazados (`0`, `-1`, `abc`, vacío,
     `1,555`, `10000`, `9999,995`) con la regla de HU-22 y el valor anterior intacto;
   - historia de un Sprint finalizado: rechazada y sin cambios; si el Sprint se finaliza entre la lectura
     y el guardado, también;
   - historia inexistente o de otro proyecto: rechazada;
   - guardar el mismo valor: aceptado.
2. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - guardado válido: 303 a `/estimaciones`, "Horas estimadas registradas" o "Horas estimadas
     actualizadas" una sola vez y valor mostrado con coma; con dos proyectos, solo en el activo;
   - valor inválido y por encima del máximo: mensaje junto al campo de esa fila, lo escrito conservado y
     el valor anterior igual;
   - historia de un Sprint finalizado con un envío directo: 409 y sin cambios;
   - historia no numérica, inexistente o de otro proyecto: "La historia no existe"; un campo de proyecto
     enviado se ignora;
   - falla de guardado: "La operación no pudo completarse", sin éxito y valor anterior intacto;
   - recarga después de guardar: no se repite el guardado ni el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada cambia.
3. Vista:
   - título, estado, Story Points, horas estimadas y horas reales de cada historia, solo del proyecto
     activo y las más recientes primero;
   - sin estimar: "Sin estimar" y nunca 0; sin horas reales: 0;
   - historia de un Sprint finalizado con el campo deshabilitado y el motivo "Sprint finalizado";
   - sin historias: "Aún no hay historias";
   - título con `<script>` visto como texto; falla de lectura: página de error de la aplicación.
4. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.
5. PostgreSQL: `PostgreSQLEstimateStore` y `PostgreSQLEstimateCatalog` contra PostgreSQL real con fixtures
   propias.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas
ejecutan las reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para
simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/017-story-estimates/`. En PowerShell, ejecutar `chcp 65001` antes para
que las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/017-story-estimates/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/017-story-estimates/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/017-story-estimates/cobertura.txt
```

## Manual breve de usuario: pantalla "Estimaciones"

La pantalla "Estimaciones" (`/estimaciones`) permite registrar y modificar las horas estimadas de las
historias del proyecto activo, para compararlas después con las horas reales. Si no hay un proyecto
elegido, la aplicación pide elegirlo primero.

**Qué se ve**: una fila por historia, las más recientes primero, con el título, el estado, los Story
Points, las horas estimadas y las horas reales registradas. Lo que todavía no se estimó aparece como "Sin
estimar"; las horas reales aparecen como 0 si nadie registró horas.

**Registrar o modificar**: escribir las horas en el campo de la fila y pulsar "Guardar". Se acepta coma o
punto como separador decimal (`1,5` o `1.5`), con hasta dos decimales y hasta 9999,99 horas; no se usan
separadores de miles. El primer valor muestra "Horas estimadas registradas" y los siguientes "Horas
estimadas actualizadas". Las horas siempre se muestran con coma.

**Historias bloqueadas**: las historias de un Sprint finalizado muestran el campo deshabilitado con el
motivo "Sprint finalizado"; sus horas estimadas ya no se pueden cambiar.

**Mensajes de error**:
- "El valor de horas no es válido": el valor es 0, negativo, vacío, no numérico o tiene más de dos
  decimales.
- "Las horas estimadas no pueden superar 9999,99".
- "La historia no existe": la historia no es del proyecto activo.
- "La operación no pudo completarse": no se guardó nada; volver a intentarlo más tarde.

Ante un error, lo escrito queda en el campo para corregirlo y el valor anterior no cambia. Cambiar las
horas estimadas no modifica las horas reales ni ningún otro dato de la historia.
