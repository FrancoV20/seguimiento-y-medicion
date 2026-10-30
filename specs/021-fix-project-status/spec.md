# Feature Specification: FIX-HU-21 / Estado "Activo" al crear un proyecto

**Feature Branch**: `FIX-HU-21`

**Created**: 2026-10-29

**Status**: Draft

**Input**: Bug detectado al usar la pantalla Proyectos (HU-21): un proyecto recién creado no admite historias en el Product Backlog (HU-02/HU-22).

## Por qué se hace

- `NewProject` (HU-01, `src/project/project.go`) crea el proyecto con estado "Active", pero la base (`DEFAULT 'Activo'`), el Product Backlog (`ProjectIsActive`) y los defectos (HU-06) consideran activo solo al proyecto con estado "Activo".
- Resultado: todo proyecto creado desde la aplicación queda rechazado como "no existe o no está activo" al cargarle historias.
- Los tests de HU-02 no lo detectaron porque insertan el proyecto directamente en la base con "Activo"; los de HU-01 esperaban "Active".

## Acceptance Scenarios

Los escenarios de HU-01 y HU-21 que no se nombran acá no cambian.

**Escenario modificado (HU-01): Alta exitosa.**
Dado un nombre, fechas válidas y al menos un integrante existente,
Cuando el Scrum Master crea el proyecto,
Entonces el proyecto se registra con estado "Activo".

**Escenario nuevo: Un proyecto recién creado admite historias.**
Dado que el Scrum Master creó un proyecto desde la aplicación,
Cuando agrega a ese proyecto una historia con datos válidos,
Entonces la historia se registra en el Product Backlog en estado "Pendiente".

**Escenario nuevo: Proyectos guardados antes del fix.**
Dado un proyecto guardado con estado "Active" antes de este fix,
Cuando se aplican las migraciones,
Entonces el proyecto queda en estado "Activo" y admite historias.

## Requirements

- **FR-001**: Todo proyecto nuevo MUST nacer con estado "Activo".
- **FR-002**: El valor "Activo" MUST estar definido en un solo lugar del paquete `project` (constante `StatusActive`).
- **FR-003**: Una migración nueva (último número de `main` + 1) MUST pasar a "Activo" los proyectos guardados como "Active". No se edita ninguna migración existente.