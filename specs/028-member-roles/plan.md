# Plan de implementación: HU-28 – Roles del Proyecto (mínimos)

**Rama**: `HU-28` | **Fecha**: 2026-10-28 | **Spec**: [spec.md](spec.md)
**Entrada**: `specs/028-member-roles/spec.md`
**Constitución**: v1.1.0

## Resumen

HU-28 agrega el rol "Scrum Master" o "Integrante" por proyecto y reserva al Scrum Master 9 rutas POST: planificar y
cerrar Sprints, modificar el proyecto y conducir el Planning Poker.

- Un paquete nuevo, `src/role`, con la regla pura (`Role`, `ParseRole`, `CanManage`) y el puerto `Store` con su
  adaptador PostgreSQL, que usa el comando.
- Una migración que agrega `project_members.role`.
- La consulta del contexto de HU-20b suma el rol del integrante y si el proyecto tiene Scrum Master; los dos viajan en
  `ActiveContext`.
- `Server.RequireScrumMaster(h)` y `Server.RequireScrumMasterOfPathProject(h)` (para HU-11) envuelven las rutas al
  registrarlas, sin cambiar los handlers de las otras historias.
- La estructura común muestra "Nombre (Rol)".
- Un comando nuevo, `src/cmd/set-role`.
- Las pruebas existentes de esas acciones se identifican como Scrum Master con un ayudante común.

`src/cmd/server/main.go` no cambia.

## Contexto técnico

**Lenguaje/Versión**: Go 1.27.1.
**Dependencias**: biblioteca estándar (`net/http`, `html/template`, `database/sql`, `flag`, `errors`, `strings`) y
`github.com/jackc/pgx/v5/stdlib`, que ya está. `go.mod` no cambia.
**Almacenamiento**: PostgreSQL. Una migración nueva (`project_members.role`) con up y down. HU-28 lee `project_members`
en la consulta del contexto, y el comando escribe `project_members.role`.
**Pruebas**:

- unitarias de `src/role` (`ParseRole`, `CanManage`) y de `RequireScrumMaster` con un contexto armado a mano;
- integración contra PostgreSQL real:
  - `ResolveContext` con el rol;
  - `role.PostgreSQLStore`;
  - la migración up y down en una base descartable (con el `CHECK`);
  - el comando completo;
- handlers con `httptest` y `cookiejar`: una tabla de rutas protegidas y libres con las secciones reales y dobles de
  las acciones que cuentan llamadas;
- BDD en `test/web_member_roles_bdd_test.go`, con el cableado de `main.go` y los datos del Issue;
- el comando: `assignRole` con un doble de `role.Store` y `run` para las opciones y el entorno (sin base), más una
  prueba de integración.

Se corren con `go test -count=1 -p 1 ./...`.
**CI**:

- `gofmt -l .` sin salida y `go vet ./...` limpio;
- `migrate up` antes de las pruebas;
- `go test ./... -p 1 -race` con cobertura.

Los archivos nuevos se formatean; los existentes se editan a mano respetando su fin de línea (CRLF en la copia de
Windows).
**Plataforma**: servidor web local y comando de consola, en Windows y Linux (CI).
**Tipo de proyecto**: aplicación web renderizada en el servidor, sin JavaScript, más un comando.
**Objetivos de rendimiento**: la validación del contexto con el rol sigue por debajo de 50 ms
(`TestPostgreSQLContextStore_ResolveContext_MenosDe50ms`, SC-006).
**Restricciones**:

- Integrar después de HU-27, al final de todo.
- No cambiar el interior de los handlers de HU-11, HU-15, HU-23, HU-05a, HU-05b ni HU-18.
- No tocar `main.go`.

**Escala**: proyectos del curso (decenas de integrantes).

## Verificación de la constitución

| Principio | Cómo se cumple | Estado |
|---|---|---|
| I. Alcance guiado por valor | Could Have, valor 55. Cada FR sale de un escenario del Issue, de una decisión del equipo o de la respuesta de clarify (FR-016). Product Owner, pantalla de roles y botones quedan fuera. | PASA |
| II. Calidad por construcción | Regla pura en `src/role`. Un solo envoltorio para las 9 rutas, aplicado al registrar, sin copiar lógica en cada handler. El rol viaja en la consulta que ya existe. Se reutilizan `RenderError`, `SetFlash`, `firstSegment`, `parseID`, `LoadConfig` y los textos de HU-20b. | PASA |
| III. Primero las pruebas | 13 escenarios en Dado/Cuando/Entonces. Cada bloque sigue RED → `evidencia-red.txt` → GREEN. Las pruebas existentes se ven fallar con el envoltorio (RED) antes de actualizarlas. | PASA |
| IV. Trabajo ágil y colaborativo | Se integra después de HU-27, al final (nota 7). T003 avisa qué archivos de HU-05a, HU-05b, HU-11, HU-15, HU-18, HU-20a, HU-20b y HU-23 se tocan; T035 lo repite en el PR. | PASA |
| V. Medición | No agrega métricas; evita que un integrante cierre un Sprint por error y distorsione la velocidad. | PASA (no aplica directamente) |
| VI. Persistencia PostgreSQL | `role.Store` tiene adaptador PostgreSQL y prueba de integración. La ampliación de `ResolveContext` se prueba contra PostgreSQL. La migración se prueba up y down. | PASA |

## Estructura

### Documentación

```text
specs/028-member-roles/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
├── evidencia-red.txt        (durante la implementación)
├── evidencia-pruebas.txt    (cierre)
└── cobertura.txt            (cierre)
```

No hay carpeta `contracts/`. Las rutas, las respuestas, la consulta y el contrato del comando están en
[data-model.md](data-model.md).

### Código

Archivos nuevos:

```text
src/role/role.go                              Role, RoleScrumMaster, RoleMember, ParseRole, CanManage
src/role/role_test.go
src/role/store.go                             Store, ErrProjectNotFound, ErrMemberNotInProject
src/role/postgres_store.go                    PostgreSQLStore
src/role/postgres_store_integration_test.go
db/migrations/<NNN>_add_project_member_role.up.sql
db/migrations/<NNN>_add_project_member_role.down.sql
db/member_role_migration_integration_test.go
src/web/scrum_master.go                       textos, RequireScrumMaster, RequireScrumMasterOfPathProject
src/web/scrum_master_test.go                  envoltorio con contexto armado a mano
src/web/scrum_master_routes_test.go           tabla de rutas protegidas y libres
src/web/member_role_helpers_test.go           setTestScrumMaster
src/web/member_role_context_test.go           "Nombre (Rol)" en la estructura común
src/cmd/set-role/main.go                      main, run, assignRole
src/cmd/set-role/main_test.go
test/web_member_roles_bdd_test.go             escenarios del Issue
test/member_role_helpers_test.go              configurarScrumMaster
```

Archivos existentes que se editan, por historia:

| Historia | Archivo | Cambio |
|---|---|---|
| HU-20b | `src/web/context.go` | `ResolvedContext` suma `MemberRole` y `HasScrumMaster` |
| HU-20b | `src/web/render.go` | `ActiveContext` suma los mismos dos campos |
| HU-20b | `src/web/postgres_context_store.go` | `resolveContextQuery` con `pm.role` y `EXISTS` |
| HU-20b | `src/web/context_middleware.go` | copia los dos campos a `ActiveContext` |
| HU-20b | `src/web/postgres_context_store_integration_test.go` | casos del rol |
| HU-20a / HU-27 | `src/web/templates/layout.html` | "Nombre (Rol)" |
| HU-23 | `src/web/sprints_handlers.go` | envuelve 3 rutas |
| HU-23 | `src/web/sprints_handlers_test.go`, `test/web_sprints_bdd_test.go` | identificación como Scrum Master (dobles y fixtures reales) |
| HU-05a / HU-05b / HU-18 | `src/web/poker_handlers.go` | envuelve `abrir`, `cerrar`, `nueva-ronda` y `estimacion` |
| HU-05a | `src/web/poker_handlers_test.go`, `src/web/poker_view_test.go`, `test/web_poker_bdd_test.go` | quien abre la ronda es Scrum Master |
| HU-11 (T001) | `src/web/projects_handlers.go` | envuelve `POST /proyectos/{id}/modificar` |
| HU-15 (T001) | `src/web/close_handlers.go` | envuelve `POST /cierre` |
| HU-11, HU-15, HU-05b, HU-18 (T001) | sus pruebas de handler y BDD | identificación como Scrum Master |
| Documentación | `README.md` | comando `set-role` y asignar un Scrum Master por proyecto después de migrar |

No se tocan `main.go`, `RequireContext`, `Server.Register`, los handlers por dentro ni las migraciones existentes.

## Decisiones de diseño

Detalle y alternativas en [research.md](research.md).

1. Paquete `src/role` (no `src/project`) para no chocar con el `ErrProjectNotFound` de HU-11.
2. `ParseRole` exacto con recorte de bordes; `CanManage` solo para "Scrum Master".
3. El rol y `HasScrumMaster` se leen en la misma consulta del contexto (`EXISTS` sobre la PK).
4. `RequireScrumMaster` es un método de `Server` (necesita el `Renderer` para el 403); se aplica dentro de `Routes`,
   detrás de `RequireContext`.
5. HU-11: `RequireScrumMasterOfPathProject` exige que el `{id}` sea el proyecto activo (respuesta de clarify).
6. Sin integrante, la redirección vuelve a `/<primer segmento>` de la ruta, que coincide con la sección en las 9.
7. Título propio del 403 ("Acción reservada al Scrum Master"); el de HU-20b es para otro origen.
8. Comando: `run` (opciones, rol, entorno) y `assignRole` (lógica sobre el puerto). Códigos 0, 1 y 2. Orden: rol,
   proyecto, integrante.
9. Ayudantes comunes `setTestScrumMaster` y `configurarScrumMaster`, que reutilizan los de HU-27; los dobles de HU-23
   devuelven un Scrum Master.
10. Número de la migración: la mayor en `main` + 1, después de la de HU-27.

## Contratos que se consumen

- **HU-20b**:
  - `ContextStore.ResolveContext`, `ResolvedContext`, `ActiveContext`, `contextMiddleware`;
  - `RequireContext`, `firstSegment`, `parseID`, `SetFlash`;
  - `MsgProjectRequired`, `MsgMemberRequired`, `pathSelectProject`, `pathSelectMember`.
- **HU-20a**: `Server`, `Routes`, `Renderer.RenderError`, `layout.html`, `LoadConfig`.
- **HU-26**: `SignedContextValue` (dobles de prueba).
- **HU-27 (T001: HU-27)**:
  - la identificación con contraseña;
  - `setTestPassword`, `configurarClave`, `claveDePrueba`;
  - `layout.html` con "Salir";
  - el patrón del comando `set-password`.
- **HU-23**: `RegisterSprints` y sus 3 rutas POST.
- **HU-05a**: `RegisterPlanningPoker` y `POST /planning-poker/abrir`.
- **HU-05b, HU-18, HU-11, HU-15 (T001)**: las rutas de la tabla de `data-model.md`.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| HU-27 no está en la rama base | T001 detiene todo. |
| Alguna de HU-11, HU-15, HU-05b o HU-18 no está en `main` al implementar | T001 la anota; su ruta se envuelve en la rama de esa historia (aviso de T003) y la tabla de rutas de prueba se completa allí. |
| Al migrar, ningún proyecto tiene Scrum Master y nadie puede planificar | Quickstart, README y aviso en el PR: asignar un Scrum Master por proyecto con `set-role` después de migrar (y configurar las contraseñas de HU-27). |
| HU-11 permite quitar al único Scrum Master | Fuera de alcance; el mensaje del 403 indica el comando para resolverlo. |
| Rompe las pruebas de HU-23, HU-05a y de las historias que entren antes | Se integra al final. T001 cuenta las pruebas. El Bloque E las actualiza con ayudantes comunes; no se sube la rama entre T024 y T032. |
| Los dobles de `ContextStore` de HU-23 no pasan por la contraseña | Llevan la cookie firmada del integrante (`SignedContextValue`) y su `ResolveContext` lo devuelve como Scrum Master. |
| El `EXISTS` agrega costo a cada solicitud | Filtra por la primera columna de la PK; la prueba de 50 ms de HU-20b lo controla (SC-006). |
| Número de migración ocupado por HU-27 u otra | T001 lo fija después de HU-27; T033 lo reconfirma antes del PR. |
| Conflictos en `poker_handlers.go`, `sprints_handlers.go`, `layout.html` y `context_middleware.go` | Cambios de una línea por ruta y dos campos en el contexto; se integra al final. |
| Los botones siguen visibles y un integrante recibe un 403 al usarlos | Decisión del equipo (nota 6); el mensaje explica por qué. Deshabilitarlos queda como mejora futura. |
| Fin de línea CRLF en archivos existentes | Se editan a mano, sin `gofmt -w`. |
| "Planificación de proyecto.pdf" no está en el repositorio | Se usaron el Issue y las notas; si dice otra cosa, prevalece y se corrige la spec antes de T002. |

## Verificación posterior al diseño

Vuelvo a revisar el diseño cerrado contra la constitución:

- **Principio VI**: `role.Store` y la consulta del contexto ampliada se prueban contra PostgreSQL (T007, T013). La
  migración se prueba up y down (T006).
- **Principio II**: la regla está en un solo lugar (`role.CanManage`), y el control en un solo envoltorio; los handlers
  de las otras historias no se duplican ni se editan por dentro.
- **Principio III**: los bloques siguen RED → evidencia → GREEN. En el de pruebas existentes, el RED es la suite
  fallando con el envoltorio.

Sigue en **PASA**, sin violaciones que justificar.
