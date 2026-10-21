# Tasks: FIX-HU-04 / Un solo Sprint activo por proyecto y estado "En Sprint"

**Input**: `specs/004-fix-single-active-sprint/spec.md`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar. Se ejecutan con `go test -count=1 -p 1 ./...`.

**Migraciones**: ver la tarea correspondiente; el número es el último de `main` más uno. Nadie edita una migración existente.

## Tareas

Las pruebas se escriben antes que el código (RED → GREEN → REFACTOR).

- [ ] T001 Revisar HU-04 en `main`: paquete `src/sprint`, firmas de `CreateSprint`, `AssignStories` y `RemoveStory`, cómo y dónde se incrementa `projects.next_sprint_number`, qué estado queda en la historia al asignar (confirmar que es "En Sprint") y qué estado recibe al quitarla (confirmar "Pendiente"), mensajes y tipos de error. Buscar los tests de HU-04, HU-07, HU-03 y de cualquier otra historia que creen más de un Sprint activo en el mismo proyecto. Anotar el resultado en `specs/004-*/data-model.md`.
- [ ] T002 [P] Escribir las pruebas del servicio para los escenarios nuevos: segundo Sprint rechazado con el número del activo, rechazo sin saltos de numeración, creación con todos los anteriores finalizados, regla por proyecto. Para dejar un Sprint finalizado en las pruebas, cambiar su estado directamente en la base (HU-15 todavía no existe).
- [ ] T003 [P] Escribir la prueba de integración contra PostgreSQL de la creación simultánea (dos goroutines que crean un Sprint para el mismo proyecto): se crea uno solo, el otro recibe el error de Sprint activo y los números no tienen huecos.
- [ ] T004 [P] Escribir las pruebas de los dos escenarios reescritos (asignar una historia ya asignada, quitar y volver a asignar) y verificar los estados "En Sprint" al asignar y "Pendiente" al quitar.
- [ ] T005 Implementar la regla en el servicio y en el almacenamiento PostgreSQL de Sprints, como en "Cómo debe resolverse la regla", y definir el error con el número del Sprint activo.
- [ ] T006 Ajustar el mensaje del escenario de historia ya asignada para que indique "ya está asignada al Sprint N" y no "a otro Sprint en curso".
- [ ] T007 Corregir los tests de T001 que creaban dos Sprints activos en el mismo proyecto: dejar el primero como finalizado antes de crear el segundo, o usar otro proyecto.
- [ ] T008 Actualizar la documentación: `specs/004-*/spec.md` (escenarios nuevos y reescritos, requisito de un solo Sprint activo, estados "Pendiente / En Sprint / En progreso / Hecho"), `quickstart.md`, `tasks.md`, y el texto de HU-04 en el documento de planificación y en el Issue de HU-04.
- [ ] T009 Ejecutar `go test -count=1 -p 1 ./...` completo y guardar la salida en `specs/004-*/evidencia-fix-sprint-activo.txt`, junto con la salida inicial con las pruebas nuevas fallando (`evidencia-fix-sprint-activo-red.txt`).
- [ ] T010 Pull Request desde `FIX-HU-04` revisado por otro integrante. Al integrarlo, avisar a quien tome HU-23, HU-14 y HU-24.

## Notes

- No se edita ninguna migración existente. Este fix no necesita migraciones; si surgiera la necesidad, el número es el último de `main` más uno.
- Los nombres de error y mensajes los usa HU-23, así que dejalos documentados en `data-model.md` de HU-04.
- Una historia que terminó en "Hecho" en un Sprint finalizado sigue asignada a ese Sprint: no se puede volver a asignar. Eso es correcto y lo cubre el escenario de historia ya asignada.

## Definition of Done

- [ ] Los tests nuevos se escribieron primero y el historial de commits lo muestra.
- [ ] `go test ./...` completo, `go vet` y `gofmt` en verde en el pipeline.
- [ ] La integración contra PostgreSQL prueba la creación simultánea.
- [ ] Los escenarios se verificaron a mano con la aplicación o, mientras HU-23 no esté integrada, con las pruebas de integración del servicio.
- [ ] Spec, quickstart y tasks de HU-04 actualizados; Issue de HU-04 con el texto nuevo.
- [ ] Pull Request aprobado por otro integrante y rama integrada a `main` sin conflictos.
