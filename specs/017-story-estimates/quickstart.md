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
   - recarga después de guardar: no se repite ni repite el mensaje;
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