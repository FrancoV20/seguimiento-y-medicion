# T001: Reconocimiento del modelo de datos para FIX-HU-01

## Rama y referencia revisadas

- Rama de trabajo activa: `Fix-HU-01` (la escritura real difiere en mayúsculas de `FIX-HU-01`).
- `main` revisada en `901f9c0` (`Creacion de md de HU-23`), sin cambios locales al comenzar T001.
- La implementación HU-01 y la migración `001` del árbol de trabajo coinciden con las de `main`.

## Validación y mensajes existentes

- `src/project/project.go`, función `NewProject`: rechaza fechas cero y fechas de fin anteriores mediante `endDate.Before(startDate)`. Por eso actualmente permite fechas iguales.
- `src/project/service.go`, función `(*ProjectService).CreateProject`: delega la validación de fechas a `NewProject`; no contiene otra comparación del rango.
- `src/project/errors.go`: `ErrInvalidPeriod` tiene actualmente el mensaje `end date cannot be before start date`.
- La regla se implementará en una sola ubicación de dominio (`NewProject`), usada por el servicio. El mensaje de error de dominio se actualizará en T005; el mensaje de éxito y los mensajes de nombre e integrantes quedan fuera del cambio.

## Restricción de base de datos

- `db/migrations/001_create_projects.up.sql` declara `CONSTRAINT chk_project_dates CHECK (end_date >= start_date)`.
- `main` tiene como número de migración más alto `007` (`007_create_sprint_metrics`); el número siguiente disponible para este fix es `008`. Hay saltos en la secuencia, pero se toma el número más alto, no el recuento de archivos.
- PostgreSQL está ejecutándose en Docker, pero no fue posible ejecutar `\d projects`: la conexión local devolvió `FATAL: role "sym_user" does not exist`. Por lo tanto, el nombre confirmado en esta fase es el definido por la migración versionada (`chk_project_dates`); el nombre efectivo de la base local no pudo verificarse.

## Tests y fixtures que actualmente dependen de fechas iguales

1. `src/project/service_test.go`, `TestProjectService_CreateProject_SuccessWithEqualDates`: arma una solicitud con `StartDate` y `EndDate` iguales y espera que se cree correctamente. Es el caso unitario que debe cambiar de expectativa en T002.
2. `src/metrics/postgres_metrics_store_integration_test.go`, `TestPostgreSQLMetricsStoreLoadsAndSavesMetrics`: inserta un proyecto con `CURRENT_DATE` tanto para `start_date` como para `end_date`. Es un fixture de otra historia que deberá usar una fecha de fin posterior en T007.

La búsqueda de `StartDate`, constructores `NewProject` e inserciones SQL en los archivos Go no encontró otros proyectos creados con fechas iguales. Los casos de proyecto con fin anterior en `src/project/service_test.go` y `test/project_bdd_test.go` son inválidos deliberadamente; los fixtures de backlog, Sprint y almacenamiento de proyectos usan fechas de fin posteriores.
