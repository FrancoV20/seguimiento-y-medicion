# Tareas: HU-28 – Roles del Proyecto (mínimos)

**Entrada**: `specs/028-member-roles/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Requisitos previos**: HU-27 en la rama base (T001 detiene todo si no está). HU-11, HU-15, HU-05b y HU-18 para
cubrir todas las rutas. Se integra al final de todo.

Formato: `[ID] [P] [E#] descripción`
- **[P]**: se puede hacer en paralelo (archivos distintos, sin dependencia).
- **[E#]**: escenario del Issue que cubre.
- Cada bloque sigue RED → GREEN: las pruebas se escriben y se ven fallar antes del código, y la salida se guarda en
  `specs/028-member-roles/evidencia-red.txt`, en la sección del bloque.
- Antes de cada ejecución en PowerShell: `chcp 65001`.
- Datos: "Datos de ejemplo" de `spec.md` y "Datos de prueba" de `data-model.md`. Las contraseñas se configuran con los
  ayudantes de HU-27 (T001: HU-27).

## Fase 1: Preparación

- [ ] T001 Hacer `git pull` de `main` y confirmar nombres y rutas. Anotar el resultado en `research.md` ("Resultado de
  T001").
  1. **HU-27**: comprobar que está en la rama base
     (`git grep -n "RegisterContextWithLockout\|password_hash\|setTestPassword\|configurarClave"`). Si no está,
     **detener todo y avisar**.
  2. **Rutas**: listar las rutas POST reales con `git grep -n '"POST ' src/web`. Confirmar las 9 de la tabla "Rutas
     protegidas" de `spec.md`, los nombres de sus handlers y de las funciones de registro de HU-11 y HU-15, y el
     valor de ruta `{id}` de HU-11. Anotar las historias que todavía no están.
  3. **Pruebas que hacen POST a esas rutas**: `git grep -n` de cada ruta en `'*_test.go'` y `test/`; hoy son 5
     archivos de HU-23 y HU-05a.
  4. **Dobles**: los que implementan `ContextStore` completo.
  5. **Migración**: la mayor en `db/migrations`. Fijar `<NNN>` = mayor + 1 y reemplazarlo en `data-model.md` y
     `plan.md`.
- [ ] T002 Crear la rama `HU-28` desde `main` actualizado, solo si no hay cambios sin commitear.
- [ ] T003 Avisar al equipo qué archivos se tocan:
  - `context.go`, `render.go`, `postgres_context_store.go`, `context_middleware.go`, `layout.html`;
  - `sprints_handlers.go`, `poker_handlers.go`, `projects_handlers.go` y `close_handlers.go`;
  - las pruebas de T001.

  A las historias que falten en T001, pedirles que envuelvan su ruta con `RequireScrumMaster` al integrarse
  después.
- [ ] T004 Línea base: `go test -count=1 -p 1 ./...` en verde antes de empezar.

## Fase 2: Bloque A – Dominio y migración (`src/role`, `project_members.role`)

**Objetivo**: la regla pura, la columna y el adaptador que usa el comando.

**Prueba independiente**: `go test ./src/role/... ./db/...` con `DATABASE_URL`.

### Pruebas (RED)

- [ ] T005 [P] [E11] `TestParseRole` y `TestCanManage` en `src/role/role_test.go`, con las tablas de `data-model.md`
  ("Scrum Master" con espacios en los bordes sí; "scrum master", "SM", "Product Owner" y `""` no).
- [ ] T006 [P] `TestMigracion_ProjectMemberRole_UpYDown` en `db/member_role_migration_integration_test.go`, sobre una
  base descartable, con el patrón de `TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable`:
  - las `up` hasta `<NNN>` dejan una fila previa de `project_members` con 'Integrante';
  - `INSERT` sin `role` → 'Integrante'; `UPDATE` a 'Product Owner' → error del `CHECK`;
  - la `down` quita la columna y la `up` vuelve a aplicarse.
- [ ] T007 [P] [E10, E11] `TestPostgreSQLStore` en `src/role/postgres_store_integration_test.go`, con «Alfa» (Ana,
  Beto, Caro) y «Beta» (Dani):
  - `ProjectName` de «Alfa» y de un proyecto inexistente (`ErrProjectNotFound`);
  - `SetMemberRole` de Beto a "Scrum Master" devuelve "Beto" y cambia la fila;
  - Dani en «Alfa» y un integrante inexistente → `ErrMemberNotInProject`, sin cambios;
  - base cerrada → error distinto de los dos.
- [ ] T008 Ejecutar T005–T007, confirmar que fallan (no existen el paquete ni la migración) y guardar la salida en
  `evidencia-red.txt` (sección "Dominio y migración").

### Implementación (GREEN)

- [ ] T009 Implementar `src/role/role.go`: `Role`, `RoleScrumMaster = "Scrum Master"`, `RoleMember = "Integrante"`,
  `ErrInvalidRole`, `ParseRole` (`TrimSpace` y coincidencia exacta) y `CanManage`.
- [ ] T010 Crear `db/migrations/<NNN>_add_project_member_role.up.sql`
  (`ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'Integrante' CHECK (role IN ('Scrum Master', 'Integrante'))`) y
  `.down.sql` (`DROP COLUMN IF EXISTS role`). Aplicarla en la base local.
- [ ] T011 Implementar `src/role/store.go` (`Store`, `ErrProjectNotFound`, `ErrMemberNotInProject`) y
  `src/role/postgres_store.go` (`NewPostgreSQLStore`, `ProjectName` y `SetMemberRole` con `UPDATE … FROM members …
  RETURNING display_name`). Un identificador fuera de 1..2147483647 → el error que corresponde, sin consultar.
- [ ] T012 Ejecutar T005–T007 en verde.

## Fase 3: Bloque B – Contexto con rol (escenario 1)

**Objetivo**: cada solicitud conoce el rol del integrante identificado y si el proyecto tiene Scrum Master; la
estructura común muestra "Nombre (Rol)".

**Prueba independiente**: integración de `ResolveContext` y una página con Ana y con Beto identificados.

### Pruebas (RED)

- [ ] T013 [P] [E1, E9] `TestPostgreSQLContextStore_ResolveContextConRol` en
  `src/web/postgres_context_store_integration_test.go`: los 5 casos de la tabla de `data-model.md` (`MemberRole` y
  `HasScrumMaster`). `TestPostgreSQLContextStore_ResolveContext_MenosDe50ms` sigue sin cambios y en verde (SC-006).
- [ ] T014 [P] [E1] Crear `src/web/member_role_helpers_test.go` con `setTestScrumMaster(t, db, projectID, memberID)`
  (`setTestPassword` de HU-27 más `UPDATE project_members SET role = 'Scrum Master'`). Escribir
  `TestContexto_RolEnLaEstructuraComun` en `src/web/member_role_context_test.go`, con `httptest` y `cookiejar`:
  - identificada Ana (Scrum Master), una página muestra "Ana (Scrum Master)";
  - identificado Beto, "Beto (Integrante)";
  - sin integrante, no aparece "(" junto a "Integrante:".
- [ ] T015 Ejecutar T013–T014, confirmar que fallan (faltan los campos y la columna en la consulta) y guardar la salida
  en `evidencia-red.txt` (sección "Contexto con rol").

### Implementación (GREEN)

- [ ] T016 Agregar el rol al contexto.
  - Sumar `MemberRole role.Role` y `HasScrumMaster bool` a `ResolvedContext` (`src/web/context.go`) y a
    `ActiveContext` (`src/web/render.go`).
  - Ampliar `resolveContextQuery` y su `Scan` en `src/web/postgres_context_store.go` (`pm.role` con
    `sql.NullString` y el `EXISTS`).
  - En `src/web/context_middleware.go`, copiar `HasScrumMaster` cuando el proyecto existe y `MemberRole` cuando
    completa al integrante; un valor que `ParseRole` no reconoce queda como `RoleMember`.
  - En `src/web/templates/layout.html`, mostrar `{{.MemberName}} ({{.MemberRole}})`.
- [ ] T017 Ejecutar T013–T014 en verde, junto con las pruebas de contexto de HU-20b, HU-26 y HU-27
  (`go test -count=1 -p 1 -run "Contexto|Context|Integrante|HU20|HU26|HU27" ./src/web/... ./test/...`).

## Fase 4: Bloque C – Envoltorio y rutas (escenarios 2 a 10 y 12) 🎯 MVP

**Objetivo**: las 9 rutas quedan reservadas al Scrum Master; las demás no cambian.

**Prueba independiente**: las pruebas nuevas del envoltorio, de rutas y BDD. Las pruebas existentes de esas acciones
fallan hasta el Bloque E.

### Pruebas (RED)

- [ ] T018 [P] [E3, E5, E8, E9, E12, E13] Escribir en `src/web/scrum_master_test.go` las pruebas del envoltorio, con un
  `ActiveContext` armado a mano y un `h` que cuenta llamadas:
  - `TestRequireScrumMaster_ScrumMasterPasa`;
  - `TestRequireScrumMaster_IntegranteRechazado` (403, `TitleScrumMasterOnly`, `MsgScrumMasterOnly`);
  - `TestRequireScrumMaster_SinScrumMaster` (403, `MsgNoScrumMaster`);
  - `TestRequireScrumMaster_SinIdentificacion` (303 a `/contexto/integrante?volver_a=%2Fcierre` para
    `POST /cierre`);
  - `TestRequireScrumMaster_SinProyecto` (303 a elegir proyecto);
  - `TestRequireScrumMasterOfPathProject_OtroProyecto`: con `{id}` distinto, inválido o inexistente → 403,
    `TitleScrumMasterOnly` y `MsgScrumMasterOtherProject`, aunque sea Scrum Master del activo, y sin consultar el rol
    en otro proyecto. Con `{id}` igual al activo se aplican las reglas normales (Scrum Master pasa; "Integrante" →
    `MsgScrumMasterOnly`).

  En los rechazos `h` tiene 0 llamadas y la página no tiene detalles internos.
- [ ] T019 [P] [E3–E7, E9] Escribir en `src/web/scrum_master_routes_test.go`:
  - `TestScrumMasterRoutes_Protegidas`: registra las secciones reales (`RegisterSprints`, `RegisterPlanningPoker` y
    las de HU-11 y HU-15 que estén) con dobles de las acciones que cuentan llamadas, y el `PostgreSQLContextStore`
    real con «Alfa» y «Beta». Recorre la tabla de rutas protegidas presentes (T001): Beto → 403 "solo el Scrum
    Master", Dani en «Beta» → 403 "no tiene Scrum Master", Ana → la acción se llama. 0 llamadas en los rechazos.
  - `TestScrumMasterRoutes_AccionesLibres`: Beto en `POST /planning-poker/votar`, `POST /tablero/estado` y
    `POST /esfuerzo` llega a la acción.
- [ ] T020 [P] [E1–E10, E12, E13] Crear `test/member_role_helpers_test.go` con
  `configurarScrumMaster(t, db, proyecto, integrante)` (`configurarClave` de HU-27 más el `UPDATE`). Escribir
  `test/web_member_roles_bdd_test.go`, con el servidor armado como `main.go` (los `Register…` reales y los adaptadores
  PostgreSQL) y los datos del Issue:
  - `TestHU28_RolVisible`;
  - `TestHU28_ScrumMasterCierraSprint`: el Sprint 2 queda 'Finalizado', como indica HU-15;
  - `TestHU28_IntegranteNoCierraSprint`: sigue 'Activo';
  - `TestHU28_IntegranteNoPlanificaSprint`: no hay Sprint nuevo; «Perfil» sin Sprint; «Login» sigue asignada; el
    `GET /sprints` de Beto sigue mostrando el formulario y los botones (FR-012);
  - `TestHU28_IntegranteNoModificaProyecto`: «Alfa» sin cambios;
  - `TestHU28_ModificarOtroProyecto`: Ana, con «Alfa» activo, guarda cambios en «Beta» → el mensaje
    `MsgScrumMasterOtherProject` y «Beta» sin cambios;
  - `TestHU28_IntegranteNoConducePoker`: la ronda 1 sigue abierta y sin estimación;
  - `TestHU28_IntegranteVotaYTrabaja`;
  - `TestHU28_SinIdentificarse`;
  - `TestHU28_ProyectoSinScrumMaster`;
  - `TestHU28_AsignarRol`: Beto pasa a Scrum Master con `role.NewPostgreSQLStore`, como lo hace el comando, y cierra
    el Sprint 2 sin volver a identificarse;
  - `TestHU28_ErrorAlVerificarRol`: un store envuelto que falla en `ResolveContext` en el POST → 503 y el Sprint sigue
    'Activo'.

  Los pasos de una historia que todavía no está en `main` se agregan cuando entre (T001, T003).
- [ ] T021 Ejecutar T018–T020, confirmar que fallan por la razón esperada (no existe el envoltorio y un "Integrante"
  todavía puede hacer las acciones) y guardar la salida en `evidencia-red.txt` (sección "Envoltorio y rutas").

### Implementación (GREEN)

- [ ] T022 Implementar `src/web/scrum_master.go` con la tabla de pasos de `data-model.md`:
  - los textos `TitleScrumMasterOnly`, `MsgScrumMasterOnly` y `MsgNoScrumMaster`;
  - `(*Server).RequireScrumMaster`, con `RenderError(…, http.StatusForbidden, …)` y `volver_a` =
    `"/" + firstSegment(r.URL.Path)`;
  - `(*Server).RequireScrumMasterOfPathProject`, con `parseID(r.PathValue("id"))` comparado con `ProjectID`.
- [ ] T023 Envolver las rutas donde se registran, sin cambiar los handlers por dentro:
  - `src/web/sprints_handlers.go`: las 3 rutas POST;
  - `src/web/poker_handlers.go`: `abrir`, y `cerrar`, `nueva-ronda` y `estimacion` si están;
  - `src/web/projects_handlers.go`: `POST /proyectos/{id}/modificar` con `RequireScrumMasterOfPathProject`, si está;
  - `src/web/close_handlers.go`: `POST /cierre`, si está.

  Las rutas libres no se tocan.
- [ ] T024 Ejecutar T018–T020 en verde
  (`go test -count=1 -p 1 -run "ScrumMaster|HU28" ./src/web/... ./test/...`).

**Punto de control**: las acciones quedan reservadas. Las pruebas existentes de esas acciones quedan en rojo hasta el
Bloque E; no subir la rama entre T024 y T032.

## Fase 5: Bloque D – Comando `set-role` (escenarios 10 y 11)

**Objetivo**: asignar roles desde la consola.

**Prueba independiente**: `go test ./src/cmd/set-role/...` con y sin `DATABASE_URL`.

### Pruebas (RED)

- [ ] T025 [E10, E11] Escribir `src/cmd/set-role/main_test.go`:
  - `TestAssignRole_AsignaScrumMaster`: con un doble de `role.Store`, "Beto ahora es Scrum Master de «Alfa»" y código
    0; también "Ana ahora es Integrante de «Alfa»";
  - `TestAssignRole_Rechazos`: proyecto inexistente → "El proyecto no existe"; integrante de otro proyecto →
    "El integrante no pertenece al proyecto"; falla del store → "No se pudo acceder a la base de datos"; código 1 y
    sin cambios en el doble;
  - `TestRun_UsoYEntorno`:
    - opciones faltantes o números inválidos → uso y código 2;
    - `-rol "Product Owner"` → "El rol no es válido" y código 1, sin conectarse;
    - sin `DATABASE_URL` → el mensaje de `LoadConfig` y código 1;
  - `TestRun_AsignaRol` (integración; se omite sin `DATABASE_URL`): Beto en «Alfa» pasa a 'Scrum Master' en la base.
- [ ] T026 Ejecutar T025, confirmar que falla porque el comando no existe y guardar la salida en `evidencia-red.txt`
  (sección "Comando").

### Implementación (GREEN)

- [ ] T027 Implementar `src/cmd/set-role/main.go` según `data-model.md`:
  - `run(ctx, args, stdout, getenv)`, con `flag.NewFlagSet` en `ContinueOnError` y salida descartada;
  - el rol se valida con `role.ParseRole` antes de conectarse;
  - la base se abre con `web.LoadConfig` y `sql.Open("pgx", …)` con `PingContext`;
  - `assignRole(ctx, store, projectID, memberID, r, stdout)` consulta `ProjectName` y después `SetMemberRole`;
  - textos y códigos 0, 1 y 2.
- [ ] T028 Ejecutar T025 en verde.

## Fase 6: Bloque E – Actualización de pruebas existentes

**Objetivo**: las pruebas de las acciones reservadas se identifican como Scrum Master; sus demás expectativas no
cambian.

**Prueba independiente**: `go test -count=1 -p 1 ./...` completo.

- [ ] T029 Ejecutar `go test -count=1 -p 1 ./...` después de los Bloques C y D. Confirmar que fallan solo las pruebas de
  los archivos de T001, con 403 o redirección a identificarse. Guardar la salida en `evidencia-red.txt` (sección
  "Pruebas existentes"). Si falla otra, anotarla y revisarla antes de seguir.
- [ ] T030 [P] Actualizar las pruebas de `src/web`:
  - `sprints_handlers_test.go`: el doble `sprintHandlerContextStore` devuelve un integrante de prueba como Scrum
    Master (`MemberBelongs`, `MemberName`, `MemberRole: role.RoleScrumMaster`, `HasScrumMaster: true`), y el cliente
    lleva su cookie con `SignedContextValue(CookieMember, id)`; la prueba con `PostgreSQLContextStore` real usa
    `setTestScrumMaster` y se identifica;
  - `poker_handlers_test.go` y `poker_view_test.go`: quien abre la ronda pasa por `setTestScrumMaster`;
  - las pruebas de handler de HU-11, HU-15, HU-05b y HU-18 que liste T001.

  Las demás expectativas no cambian.
- [ ] T031 [P] Actualizar las pruebas de `test/`:
  - `web_sprints_bdd_test.go`: el doble `sprintBDDContextStore` y la parte con base real;
  - `web_poker_bdd_test.go`;
  - las BDD de HU-11, HU-15, HU-05b y HU-18 que liste T001.

  Usar `configurarScrumMaster` o el doble con un Scrum Master; las demás expectativas no cambian.
- [ ] T032 Ejecutar `go test -count=1 -p 1 ./...` completo en verde.

## Fase 7: Cierre

- [ ] T033 Antes del PR, hacer `git pull` de `main` y reconfirmar:
  - que `<NNN>` sigue siendo la mayor migración + 1 (si no, renombrar los dos archivos y la prueba de T006);
  - que no entraron rutas POST nuevas que deban protegerse ni pruebas nuevas que se identifiquen para esas acciones;
  - que los nombres de las pruebas coinciden con la trazabilidad de `spec.md`.
- [ ] T034 [P] Cierre técnico:
  - `gofmt -l` sin salida en los archivos nuevos, sin `gofmt -w` en los existentes (CRLF); `go vet ./...` limpio;
    `go test ./... -p 1 -race` en verde; pipeline de GitHub Actions en verde;
  - guardar `evidencia-pruebas.txt` (`go test -count=1 -p 1 -v ./...`) y `cobertura.txt`
    (`go test -count=1 -p 1 -cover ./...`);
  - agregar al `README.md` el comando `set-role` y el paso "después de migrar, asignar al menos un Scrum Master por
    proyecto".
- [ ] T035 Verificar a mano los 13 escenarios (`quickstart.md`). Abrir el Pull Request desde `HU-28` citando HU-28, con
  las evidencias, la lista de archivos de otras historias que se editan (`plan.md`) y el aviso para la demo. Pedir la
  revisión de otro integrante, mergear con "Create a merge commit" y pasar la tarjeta a "Hecho".

## Dependencias y orden de ejecución

- **Fase 1** primero. T001 detiene todo si HU-27 no está en la rama base.
- **Bloque A** depende de la Fase 1.
- **Bloque B** depende de T009 y T010 (tipo `role.Role` y columna).
- **Bloque C** depende de B (el envoltorio lee `ActiveContext`). T020 (`TestHU28_AsignarRol`) usa también T011.
- **Bloque D** depende de A (T009 y T011). Puede hacerse en paralelo con B y C.
- **Bloque E** depende de C: su RED es la suite fallando con el envoltorio.
- **Cierre** depende de todos.
- Dentro de cada bloque: pruebas (RED) → ejecutar y guardar evidencia → implementación (GREEN).
- No subir la rama entre T024 y T032: la suite completa está en rojo.

### Oportunidades de paralelismo

- T005, T006 y T007 entre sí.
- T013 y T014 entre sí.
- T018, T019 y T020 entre sí.
- El Bloque D con los Bloques B y C.
- T030 y T031 entre sí.

## Trazabilidad: requisitos → tareas

| Requisito | Tareas |
|---|---|
| FR-001 | T005, T006, T009, T010 |
| FR-002 | T014, T016, T020 |
| FR-003 | T019, T022, T023, T033 |
| FR-004 | T019, T020, T023 |
| FR-005 | T018, T019, T020, T022 |
| FR-006 | T013, T018, T019, T020 |
| FR-007 | T018, T020, T022 |
| FR-008 | T018, T019, T023 |
| FR-009 | T019, T020 |
| FR-010 | T013, T016, T020 |
| FR-011 | T020 |
| FR-012 | T018, T020, T022 |
| FR-013 | T007, T011, T025, T027 |
| FR-014 | T005, T007, T025, T027 |
| FR-015 | T006, T010 |
| FR-016 | T018, T020, T022, T023 |
| FR-017 | T029, T030, T031, T032 |
| SC-001 | T008, T015, T021, T026, T029, T034 |
| SC-002 | T019 |
| SC-003 | T019, T020 |
| SC-004 | T032, T034 |
| SC-005 | T006, T012 |
| SC-006 | T013, T017 |

## Estrategia de implementación

1. Fase 1 y Bloque A: la regla, la columna y el adaptador; todo sigue en verde.
2. Bloque B: el rol en el contexto y en la estructura común; todo sigue en verde.
3. Bloque D (en paralelo): el comando; ya se pueden asignar Scrum Masters.
4. Bloques C y E juntos (MVP): el control de las rutas y las pruebas existentes actualizadas, en la misma rama, antes
   de subirla.
5. Cierre y PR, al final de todo.
