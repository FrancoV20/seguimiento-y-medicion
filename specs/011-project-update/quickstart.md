# Guía rápida: HU-11 – Modificación de Proyecto (web)

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-01, FIX-HU-01, HU-03, HU-04, HU-20a, HU-20b y HU-21 en `main`, ya integradas.
- Rama: `git switch main; git pull; git switch -c HU-11`.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up` (HU-11 no agrega migraciones).
- Integrantes cargados con la semilla de HU-21: `db/seeds/members.sql`.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU11|Update|DatesChanged|MembershipChanges|SprintsLeftOutside|ProjectForEdit|Membership|LogEffort" -v ./src/project/... ./src/effort/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/project/...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base.

## Escenarios de validación requeridos

1. **Reglas puras**, sin base:
   - `DatesChanged`: con las mismas fechas, con una distinta y con la misma fecha en otra zona horaria.
   - `MembershipChanges`: altas, bajas, ambas y sin cambios.
   - `SprintsLeftOutside`:
     - acortar el fin;
     - adelantar el inicio;
     - ampliar;
     - un Sprint que ya estaba fuera;
     - un Sprint en el borde exacto;
     - un Sprint fuera por los dos lados;
     - varios Sprints afectados (gana el de menor número);
     - Sprints "Finalizado".
2. **Servicio** con `TestProjectStore`:
   - `ID` 0;
   - nombre vacío;
   - sin integrantes;
   - fin igual al inicio;
   - integrante "abc";
   - repetidos;
   - errores del almacén que pasan sin cambios;
   - falla del almacén.
3. **Adaptador** (PostgreSQL):
   - cambio de nombre;
   - cambio de fechas;
   - alta;
   - baja sin horas;
   - baja con horas en el proyecto;
   - baja con horas solo en otro proyecto;
   - integrante inexistente;
   - proyecto inexistente;
   - Sprint que queda fuera;
   - Sprint que ya estaba fuera;
   - guardado sin cambios;
   - falla a mitad sin cambios;
   - dos guardados simultáneos;
   - baja mientras se cargan horas, en los dos órdenes.
4. **HU-03**: `SaveEntry` rechaza a un integrante que ya no pertenece y espera a una baja en curso; `LogEffort`
   traduce ese rechazo a `ErrMemberRequired`.
5. **Catálogo** (PostgreSQL): `ProjectForEdit` con integrantes, sin integrantes e inexistente.
6. **Handlers**:
   - los escenarios de la spec con sus códigos (303, 422, 409, 404, 500);
   - "404 antes que 422";
   - nombre con `<script>` escapado;
   - falla de lectura;
   - recarga sin reenviar;
   - rendimiento.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`) y abrir `http://localhost:8080/proyectos`. Cada proyecto
   tiene "Modificar".
2. Crear «Alfa» (2026-03-01 a 2026-06-30) con Ana y Beto. Con "Modificar", el formulario muestra esos datos y a Ana
   y Beto marcados.
3. Elegir «Alfa» como proyecto de trabajo (`/contexto/proyecto`). Cambiar el nombre a «Alfa 2» y guardar. Tiene
   que aparecer "Proyecto modificado exitosamente", la lista con «Alfa 2» y la estructura común con «Alfa 2».
4. Borrar el nombre y guardar: "project name cannot be empty" junto al nombre, con los demás datos conservados.
5. Poner como fecha de fin la misma que la de inicio: aparece el mensaje de HU-01 junto a la fecha de fin.
6. Identificarse como Beto, registrar horas en una historia de «Alfa» con Sprint activo (HU-24) y, desde otra
   ventana, desmarcar a Beto: "No se puede quitar a Beto: tiene horas registradas en este proyecto".
7. Marcar a Carla y guardar. Carla aparece en `/contexto/integrante` para «Alfa».
8. Con el Sprint 1 del 2026-06-01 al 2026-06-14, cambiar el fin a 2026-06-10: "El Sprint 1 queda fuera de las
   nuevas fechas del proyecto" junto a la fecha de fin.
9. Ampliar el fin a 2026-07-31: se guarda.
10. Identificarse como un integrante sin horas, quitarlo desde otra ventana y navegar con la primera: aparece la
    advertencia de HU-20b y la identificación se limpia.
11. Abrir `/proyectos/999999/modificar` y `/proyectos/abc/modificar`: página 404.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/011-project-update/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/011-project-update/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/011-project-update/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Fundamentos, Datos del proyecto, Integrantes, Fechas y Sprints,
Errores), cada una antes del código de su bloque.
