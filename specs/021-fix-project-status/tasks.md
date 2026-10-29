# Tasks: FIX-HU-21 / Estado "Activo" al crear un proyecto

**Input**: `specs/021-fix-project-status/spec.md`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar. Se ejecutan con `go test -count=1 -p 1 ./...`.

**Migraciones**: el número es el último de `main` más uno. Nadie edita una migración existente.

## Tareas

- [x] T001 Revisar en `main` dónde se asigna y dónde se compara el estado del proyecto (`git grep -n "Active"` y `git grep -n "'Activo'"`). Resultado esperado: solo `src/project/project.go` usa "Active".
- [x] T002 [P] Ajustar las pruebas de HU-01 para esperar "Activo": `src/project/project_test.go`, `src/project/service_test.go`, `src/project/postgres_store_integration_test.go` y `test/project_bdd_test.go`.
- [x] T003 [P] Escribir la prueba de integración `test/project_backlog_flow_test.go`: un proyecto creado con `ProjectService` sobre PostgreSQL admite una historia con `BacklogService`.
- [x] T004 Ejecutar `go test -count=1 -p 1 ./...` y guardar la salida en rojo en `specs/021-fix-project-status/evidencia-fix-estado-red.txt`.
- [x] T005 Definir `StatusActive = "Activo"` en `src/project/model.go` y usarla en `NewProject` (GREEN).
- [x] T006 Crear la migración de datos que pasa los proyectos "Active" a "Activo" (la bajada no cambia datos).
- [x] T007 REFACTOR: revisar comentarios y nombres de las pruebas tocadas.
- [x] T008 Ejecutar `go test -count=1 -p 1 ./...` completo y guardar la salida en `specs/021-fix-project-status/evidencia-fix-estado.txt`.
- [x] T009 Documentación de HU-01: `specs/001-create-project/data-model.md` ya define el estado inicial `Activo`; el error era solo del código, no requiere cambios. El escenario nuevo queda en esta spec y en el Issue del fix.
- [ ] T010 Pull Request desde `FIX-HU-21` revisado por otro integrante.

## Notes

- La migración de este fix es la `013`: se integra antes que HU-15 por ser un bug que bloquea la carga de historias, y HU-15 pasa a la `014`.
- Las pruebas fijan el literal "Activo" a propósito: si alguien cambia la constante, el test lo detecta.

## Definition of Done

- [x] Los tests nuevos se escribieron primero y el historial de commits lo muestra.
- [x] `go test ./...`, `go vet` y `gofmt` en verde en el pipeline.
- [x] Verificado a mano: crear un proyecto desde la web, seleccionarlo y cargarle una historia.
- [] Pull Request aprobado por otro integrante y rama integrada a `main` sin conflictos.