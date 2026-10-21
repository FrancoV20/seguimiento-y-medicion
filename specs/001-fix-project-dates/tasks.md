# Tasks: FIX-HU-01 / Fecha de fin posterior a la de inicio en proyectos

**Input**: `specs/001-fix-project-dates/spec.md`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar. Se ejecutan con `go test -count=1 -p 1 ./...`.

**Migraciones**: ver la tarea correspondiente; el número es el último de `main` más uno. Nadie edita una migración existente.

## Tareas

Las pruebas se escriben antes que el código (RED → GREEN → REFACTOR).

- [x] T001 Revisar HU-01 en `main`: archivo y función donde se valida el rango de fechas, mensaje actual, nombre de la restricción `CHECK` de la tabla `projects` (`\d projects` en `psql`), y último número de migración. Buscar los tests y fixtures que crean proyectos con fecha de inicio igual a la de fin, en todo el repositorio: `Get-ChildItem -Recurse -Include *.go | Select-String "StartDate"` y revisar cada uno. Anotar el resultado en `specs/001-*/data-model.md`.
- [x] T002 [P] Escribir las pruebas unitarias de la regla: fin anterior falla, fin igual falla, fin un día posterior se crea, en el archivo de pruebas del servicio de HU-01.
- [x] T003 [P] Escribir la prueba de integración del `ProjectStore` contra PostgreSQL: guardar con fechas iguales falla por la restricción y no deja filas.
- [x] T004 [P] Actualizar los escenarios BDD de HU-01: el de fecha igual pasa a rechazo y se agrega el de un día posterior.
- [x] T005 Cambiar la validación del servicio: fin menor o igual al inicio es error de fechas, con el mensaje propuesto.
- [x] T006 Crear la migración nueva `db/migrations/NNN_projects_end_after_start.up.sql` y `.down.sql` (`NNN` = último número de `main` más uno; hoy sería la 008). No se edita la migración `001`: las migraciones tienen dueño. Contenido propuesto del `.up.sql`:
  ```sql
  DO $$
  DECLARE c text;
  BEGIN
    SELECT conname INTO c FROM pg_constraint
    WHERE conrelid = 'projects'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%end_date >= start_date%';
    IF c IS NOT NULL THEN
      EXECUTE format('ALTER TABLE projects DROP CONSTRAINT %I', c);
    END IF;
  END $$;
  ALTER TABLE projects ADD CONSTRAINT projects_end_after_start CHECK (end_date > start_date);
  ```
  Y del `.down.sql`:
  ```sql
  ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_end_after_start;
  ALTER TABLE projects ADD CONSTRAINT projects_end_not_before_start CHECK (end_date >= start_date);
  ```
  Verificar el `.up.sql` y el `.down.sql` con `migrate up`, `migrate down 1` y `migrate up` de nuevo.
- [ ] T007 Corregir los tests y fixtures que usaban fechas iguales (los de T001), cambiando la fecha de fin a una posterior. Ninguna prueba de otra historia debe depender de la regla vieja.
- [ ] T008 Actualizar la documentación: `specs/001-*/spec.md` (escenario, requisito y caso borde de fechas), `quickstart.md`, `tasks.md`, y el texto de HU-01 en el documento de planificación y en el Issue de HU-01.
- [ ] T009 Ejecutar `go test -count=1 -p 1 ./...` completo y guardar la salida final en `specs/001-*/evidencia-fix-fechas.txt`, junto con la salida inicial con las pruebas nuevas fallando (`evidencia-fix-fechas-red.txt`).
- [ ] T010 Pull Request desde `FIX-HU-01` revisado por otro integrante. Al integrarlo, avisar al equipo que corra `migrate up`.

## Notes

- Si en tu base local ya hay proyectos con fecha de inicio igual a la de fin, `migrate up` falla al crear la restricción. Corregí esas filas a mano o recreá la base con `docker compose down -v` y volvé a migrar.
- No cambies los mensajes de nombre ni de integrantes de HU-01: solo el de fechas.
- Si la regla de fechas está duplicada en el handler o en otra capa, dejá una sola copia, en el servicio.

## Definition of Done

- [ ] Los tests nuevos se escribieron primero y el historial de commits lo muestra.
- [ ] `go test ./...` completo, `go vet` y `gofmt` en verde en el pipeline.
- [ ] La migración tiene `.up.sql` y `.down.sql`, probados en ambos sentidos.
- [ ] El escenario de fecha igual se verificó a mano con la aplicación en ejecución (mientras HU-21 no esté integrada, con una prueba de integración del servicio).
- [ ] Spec, quickstart y tasks de HU-01 actualizados; Issue de HU-01 con el texto nuevo.
- [ ] Pull Request aprobado por otro integrante y rama integrada a `main` sin conflictos.
