# Research: HU-11 – Modificación de Proyecto (web)

## Datos verificados en el código (main, 2026-10-27)

- `src/project`:
  - `ProjectService` tiene solo `CreateProject(req ProjectCreationRequest)`, sin `context`.
  - El puerto `ProjectStore` es `Create(p *Project) error` y `MemberExists(id int) (bool, error)`.
  - Lo implementan `PostgreSQLProjectStore` y `TestProjectStore` (en memoria, con `ShouldFailCreate` y
    `ShouldFailMember`).
  - `NewProject(name, members, start, end)` valida en este orden: nombre vacío (`ErrEmptyName`), sin integrantes
    (`ErrMissingMembers`) y fechas (`ErrInvalidPeriod`, "fin posterior al inicio").
  - `CreateProject` convierte los identificadores de integrantes con `strconv.Atoi` y rechaza los repetidos.
- Mensajes de HU-01 (`errors.go`):
  - `ErrEmptyName` = "project name cannot be empty".
  - `ErrMissingMembers` = "the member list is required".
  - `ErrInvalidPeriod` = "La fecha de finalización debe ser posterior a la fecha de inicio".
  - HU-21 los muestra tal cual con `projectFieldError`: dos están en inglés.
- `src/web`, HU-21:
  - `RegisterProjects(s, service *project.ProjectService, catalog ProjectCatalog)` registra `Section{Name:
    "Proyectos", Path: "/proyectos"}` con `GET` y `POST /proyectos`.
  - `parseProjectForm` hace lo siguiente:
    - recorta el nombre y rechaza más de 255 caracteres (`MsgProjectNameTooLong`);
    - valida las fechas vacías (`MsgMissingStartDate`, `MsgMissingEndDate`) y el formato canónico `AAAA-MM-DD`
      (`MsgInvalidDateFormat`);
    - rechaza integrantes no numéricos o menores o iguales a 0 (`MsgInvalidMember`) e ignora los repetidos.
  - Los errores de formato se muestran todos juntos y, si hay alguno, no se llama al servicio.
  - `ProjectCatalog` tiene `ListProjects` y `ListMembers`. Los fakes de pruebas (`emptyProjectListCatalog`, etc.)
    lo embeben como interfaz.
- Fakes de `ProjectStore` fuera de `src/project` (implementan `Create` y `MemberExists` a mano):
  - `failingCreateProjectStore` en `src/web/projects_handlers_test.go`;
  - `failingBDDProjectStore` en `test/web_projects_bdd_test.go`.
- `Server.Register` rechaza una segunda sección con la misma dirección (`DuplicateSectionError`).
  `Registry.Menu` resalta "Proyectos" en cualquier subdirección de `/proyectos/`.
- `parseID` acepta solo enteros canónicos entre 1 y `MaxID` (`MaxInt32`). `Renderer.Fail(NotFound(...))` muestra el
  404 de HU-20a.
- El middleware de contexto de HU-20b resuelve en cada solicitud:
  - el nombre del proyecto activo (`resolveContextQuery`);
  - la pertenencia del integrante. Si ya no pertenece, borra la cookie y muestra `MsgMemberGone` como advertencia.
- Esquema:
  - `projects.id` es `SERIAL`, con `start_date` y `end_date` `DATE` y `CHECK (end_date > start_date)` (008).
  - `project_members` tiene PK `(project_id, member_id)`, ambos `INTEGER`, con `ON DELETE CASCADE`.
  - `effort_logs(story_id BIGINT → user_stories, member_id INTEGER → members)`, sin cascada y sin referencia a
    `project_members`.
  - `user_stories.project_id` es `BIGINT`.
  - `sprints(project_id BIGINT, number, start_date, end_date, status IN ('Activo','Finalizado'))`.
- HU-04 y HU-23 no validan que un Sprint esté dentro de las fechas del proyecto
  (`specs/023-web-sprints/checklists/requirements.md`).
- Bloqueos de Sprints: `lockProjectWithoutActiveSprint` y `nextSprintNumber` bloquean el proyecto con
  `FOR UPDATE`.
- HU-03:
  - `effort.Service.LogEffort` llama a `MemberBelongsToStoryProject` antes de `SaveEntry`, fuera de la transacción.
  - `SaveEntry` abre su propia transacción (INSERT y acumulado).
  - Un integrante que no pertenece produce `FieldError{FieldMember, ErrMemberRequired}`.
  - `TestStore.SaveErr` permite simular el error de `SaveEntry`.
- Migraciones: 001–004, 007 y 008. HU-11 no agrega ninguna.

## Decisiones

### 1. Extender el puerto `ProjectStore` con `Update`

- **Decision**: `ProjectStore` suma `Update(ctx context.Context, u ProjectUpdate) error`, implementado por
  `PostgreSQLProjectStore` y `TestProjectStore`. `ProjectService` suma `UpdateProject(ctx, req) error`.
- **Rationale**: `main.go` ya construye `ProjectService` con `PostgreSQLProjectStore` y lo pasa a
  `RegisterProjects`. Con el método nuevo en el mismo puerto, ni `main.go` ni la firma de `RegisterProjects`
  cambian.
- **Costo**: los dos fakes de HU-21 (`failingCreateProjectStore`, `failingBDDProjectStore`) suman un método
  `Update` que devuelve error. Son dos archivos de prueba ajenos con tres líneas cada uno (T003 lo coordina).
- **Alternatives considered**:
  - Un puerto y un servicio separados (`ProjectUpdateStore`, `ProjectUpdateService`) (descartada: obliga a cambiar
    la firma de `RegisterProjects`, `main.go` y las cuatro llamadas de prueba de HU-21).
  - Registrar las rutas con `Server.Handle` fuera de la sección (descartada: el usuario pidió el mismo mapa de
    rutas, y `Handle` saltea `RequireContext`).
- **`context`**: `Update` y `UpdateProject` reciben `context.Context`, como los puertos de HU-03 y HU-04. `Create`
  queda como está.

### 2. Una sola regla de fechas: `NewProject`

- **Decision**: `UpdateProject` valida nombre, integrantes y fechas llamando a `NewProject`, la misma función que
  usa el alta (FIX-HU-01, R2).
- **Rationale**: no se copia la regla. El orden de los errores (nombre, integrantes, fechas) queda igual al del alta.
- **Alternatives considered**: extraer `ValidatePeriod` (descartada: cambia HU-01 sin necesidad; `NewProject` ya es
  la función única).

### 3. Conversión de integrantes compartida

- **Decision**: extraer de `CreateProject` la conversión y el control de repetidos a `parseMemberIDs(raw []string)
  ([]int, error)` en `src/project/service.go` y usarla en las dos operaciones (T029).
- **Rationale**: principio II (la lógica compartida se extrae). Es un cambio sin efecto visible, cubierto por las
  pruebas existentes de HU-01.
- **Alternatives considered**: duplicar el bucle (descartada: principio II).

### 4. Reglas propias como funciones puras

- **Decision**: `SprintsLeftOutside` (R6), `MembershipChanges` (altas y bajas) y `DatesChanged` (R7) viven en
  `src/project/update_rules.go`, sin base de datos. El adaptador las llama dentro de la transacción.
- **Rationale**: se prueban con tablas en milisegundos y la transacción solo junta los datos. Es el patrón de HU-19.
- **Alternatives considered**: resolver R6 en SQL (descartada: la regla "estaba dentro y queda fuera" y la
  elección del campo son más claras y testeables en Go).

### 5. Transacción de guardado y orden de pasos

- **Decision**: una transacción `READ COMMITTED` con este orden (ver `data-model.md`):
  1. bloquear el proyecto con `FOR NO KEY UPDATE`;
  2. comprobar que los integrantes existen;
  3. leer la pertenencia actual;
  4. borrar las bajas;
  5. contar las horas de las bajas;
  6. revisar los Sprints si cambian las fechas;
  7. actualizar el proyecto;
  8. insertar las altas.
- **Rationale**:
  - Todo o nada (decisión 3 del equipo).
  - Las reglas R5 y R6 se revisan con el proyecto bloqueado (decisión 6).
  - Borrar antes de contar cierra la carrera con HU-24 (decisión 7).
- **`FOR NO KEY UPDATE`**: HU-11 cambia solo columnas que no son clave. Ese bloqueo choca con el `FOR UPDATE` de
  HU-04, así que crear un Sprint y acortar fechas se aplican una después de la otra. A la vez, no frena las altas
  que solo referencian al proyecto, como crear una historia (HU-02).
- **Alternatives considered**:
  - `FOR UPDATE` (descartada: frena sin necesidad las inserciones con clave foránea al proyecto).
  - `SERIALIZABLE` (descartada: solo detecta conflictos entre transacciones serializables, y HU-03 no lo es).

### 6. Carrera con la carga de horas (respuesta de Clarifications)

- **Decision**:
  - HU-11 borra la pertenencia de las bajas antes de contar sus horas, en la misma transacción.
  - `PostgreSQLEffortStore.SaveEntry` (HU-03) agrega, dentro de su transacción y antes del INSERT, una consulta
    con `FOR SHARE OF pm` sobre la fila de `project_members` del integrante en el proyecto de la historia.
  - Sin fila, `SaveEntry` devuelve `errMemberNotInProject` y no guarda nada. `Service.LogEffort` lo traduce a
    `FieldError{FieldMember, ErrMemberRequired}`, el mismo error que ya da hoy para un integrante que no pertenece.
- **Rationale**:
  - Si la carga llega primero, el `DELETE` de HU-11 espera su `COMMIT`. El conteo siguiente usa una foto nueva
    (`READ COMMITTED`), ve las horas y HU-11 deshace todo.
  - Si HU-11 llega primero, la consulta con bloqueo de HU-03 espera. Cuando HU-11 confirma, la fila ya no existe y
    la consulta no devuelve nada, así que la carga se rechaza. Si HU-11 se deshace, la carga sigue.
  - No hay ciclo de bloqueos: HU-11 no bloquea `user_stories` ni `members`, y HU-03 no bloquea `projects`.
- **Alternatives considered**:
  - Bloquear el proyecto desde HU-03 (opción B de Clarifications, descartada por el usuario).
  - Aceptar la carrera (opción C, descartada).
  - `LOCK TABLE effort_logs` desde HU-11 (descartada: no cierra la carrera, porque la comprobación de HU-03 ocurre
    antes de la transacción, y frena a todos los proyectos).
- **Archivos ajenos**: `src/effort/postgres_effort_store.go` y `src/effort/service.go`, con aviso a quien hizo
  HU-03 (T003).

### 7. Revisión de Sprints solo para los que el cambio deja fuera (respuesta de Clarifications)

- **Decision**: con las fechas actuales `[A0, B0]` y las nuevas `[A1, B1]`, un Sprint `[s, e]` bloquea si estaba
  dentro (`s >= A0 && e <= B0`) y queda fuera (`s < A1 || e > B1`).
  - Se elige el de menor número.
  - El campo es `fecha_inicio` si `s < A1`; si no, `fecha_fin`.
  - Cuentan "Activo" y "Finalizado".
- **Rationale**: respuesta B de Clarifications. Ampliar nunca se rechaza, y un Sprint que ya estaba fuera (HU-04 lo
  permite) no bloquea.
- **Alternatives considered**: revisar todos los Sprints (opción A) o solo los activos (opción C), ambas descartadas
  por el usuario.

### 8. "Cambian las fechas" se compara por día de calendario

- **Decision**: `DatesChanged` compara año, mes y día de las fechas guardadas y de las enviadas. Si ninguna cambia,
  no se leen los Sprints (R7).
- **Rationale**: las columnas son `DATE` y el formulario envía `AAAA-MM-DD`. Comparar `time.Time` directamente
  podría dar distinto por la zona horaria con que el driver devuelve el `DATE`.

### 9. Integrante con horas: se nombra al primero por nombre

- **Decision**: la consulta de horas ordena por `display_name, id` y toma una fila. El error de dominio
  `MemberHasEffortError` lleva el nombre, y la web arma el texto.
- **Rationale**: el mensaje del Issue tiene un solo nombre. El orden es el mismo que el de la lista de integrantes.

### 10. Errores del dominio con datos y textos en la web

- **Decision**:
  - Errores nuevos en `src/project/update_errors.go`: `ErrProjectNotFound`, `*MemberHasEffortError{MemberName}` y
    `*SprintOutOfRangeError{Number, Field}`.
  - Textos en `src/web/projects_form.go`, junto a los de HU-21 (FR-023): `MsgProjectUpdated`, `MsgMemberHasEffort`
    y `MsgSprintOutOfRange` como formato, más `TitleProjectEdit`.
- **Rationale**: un solo lugar para los textos de la pantalla. Los mensajes de HU-01 siguen saliendo de sus errores,
  como en HU-21.
- **Alternatives considered**: textos en el `Error()` del dominio (descartada: dos lugares para los mensajes de la
  pantalla).

### 11. Catálogo de lectura: `ProjectForEdit`

- **Decision**: `ProjectCatalog` suma `ProjectForEdit(ctx, id int64) (ProjectEdit, error)` en
  `PostgreSQLProjectCatalog`.
  - Una consulta con `LEFT JOIN project_members`, una fila por integrante, ordenada por `member_id`.
  - Sin filas, devuelve `NotFound`; si falla la base, `Unavailable`.
- **Rationale**: es la lectura del formulario. Los fakes de HU-21 embeben la interfaz, así que no se rompen.
- **Alternatives considered**:
  - `array_agg` (descartada: escanear un arreglo con `database/sql` exige un tipo de pgx o texto que hay que
    parsear).
  - Leer desde `ProjectStore` (descartada: mezcla la lectura de pantalla con el núcleo, como decidió HU-21).

### 12. El 404 se decide antes de validar

- **Decision**:
  - `GET` y `POST /proyectos/{id}/modificar` convierten `{id}` con `parseID`. Con 0 responden `Fail(NotFound)` sin
    consultar la base.
  - Después leen `ProjectForEdit`, que devuelve 404 si el proyecto no existe.
  - Recién entonces el `POST` valida el formulario. Si el proyecto se borra entre la lectura y el guardado,
    `Update` devuelve `ErrProjectNotFound` y también responde 404.
- **Rationale**: escenarios 14a y 14b, y el caso límite "404 antes que 422".

### 13. Pantalla de edición propia y enlace en la lista

- **Decision**:
  - Plantilla nueva `src/web/templates/project_edit.html`, con título "Modificar proyecto", los campos de HU-21,
    el botón "Guardar" y el enlace "Cancelar" a `/proyectos`.
  - `projects.html` suma la columna "Acciones" con el enlace "Modificar" en cada fila.
- **Rationale**: el alta y la edición tienen acciones y destinos distintos. Las plantillas se cargan con
  `//go:embed templates/*.html`, así que un archivo nuevo no requiere cambios en `render.go`.
- **Alternatives considered**: reutilizar `projects.html` con un modo de edición (descartada: mezcla dos
  formularios y complica las pruebas de HTML).

### 14. Códigos de respuesta

- **Decision**:
  - 303: éxito.
  - 422: errores de formato y de HU-01, e integrante inexistente.
  - 409: integrante con horas y Sprint fuera, porque son conflictos con el estado de otros datos, según la
    convención de HU-20b.
  - 404: proyecto inexistente.
  - 500: cualquier otro error, con `MsgOperationFailed`.
  - Si falla la lectura, `Fail` de HU-20a (500 o 503).

### 15. Rendimiento (SC-006)

- **Decision**: una prueba de handler con 50 integrantes en el sistema y 20 Sprints en el proyecto mide el `GET`
  y el `POST` con cambio de fechas contra la base local, con un umbral de 500 ms.
- **Rationale**: es el volumen esperado del curso. El umbral es el mismo que en HU-05a.

## Preguntas abiertas

Ninguna. Las dos dudas abiertas se respondieron en Clarifications (decisiones 6 y 7).
