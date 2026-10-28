# Investigación: HU-28 – Roles del Proyecto (mínimos)

## Datos verificados en `main` (`357f550`, 28/10/2026)

Verificados con `git grep -n`.

**HU-20b, contexto de trabajo**

- `src/web/render.go:18`: `ActiveContext{ProjectID, ProjectName, MemberID, MemberName}`.
- `src/web/context.go`: `ResolvedContext{ProjectExists, ProjectName, MemberBelongs, MemberName}`.
- `src/web/postgres_context_store.go:77`, `resolveContextQuery`:

  ```sql
  SELECT p.name, m.display_name
  FROM projects p
  LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.member_id = $2
  LEFT JOIN members m ON m.id = pm.member_id
  WHERE p.id = $1
  ```

  Es la única consulta por solicitud. HU-28 le suma `pm.role` y si el proyecto tiene algún Scrum Master.
- `src/web/context_middleware.go`:
  - arma `ActiveContext` en la línea 53 y completa el integrante en la línea 63;
  - `RequireContext(section)` redirige a elegir proyecto (`MsgProjectRequired`) o a `pathSelectMember`
    (`MsgMemberRequired`);
  - `returnDestination` (línea 104) usa la sección para volver después de un POST.
- `src/web/return_to.go:31`: `firstSegment(path)` ("/sprints/historias" → "sprints").
- `src/web/server.go`: `Register(section, routes)` envuelve cada handler con `RequireContext(section)` y lo registra
  en el mux. Un envoltorio aplicado dentro de `Routes` queda detrás de `RequireContext`.
- `src/web/errors.go:108`: `Renderer.RenderError(w, r, status, title, message)` es la página de error con la
  estructura común; la usa el 403 de `CrossOriginGuard` (`TitleForbidden = "Solicitud rechazada"`, `messages.go:21`).
- `src/web/postgres_context_store_integration_test.go`:
  - `TestPostgreSQLContextStore_ResolveContext`;
  - `TestPostgreSQLContextStore_ResolveContext_MenosDe50ms` (SC-012 de HU-20b, que HU-28 conserva).

**Rutas en `main`**

| Historia | Archivo | Rutas | Sección |
|---|---|---|---|
| HU-23 | `src/web/sprints_handlers.go:40-43` | `POST /sprints` (`h.create`), `POST /sprints/historias` (`h.assignStories`), `POST /sprints/historias/quitar` (`h.removeStory`) | `RequiresProject: true`, sin `RequiresMember` |
| HU-05a | `src/web/poker_handlers.go:51-53` | `POST /planning-poker/abrir` (`h.open`); `POST /planning-poker/votar` (`h.vote`) sigue libre | `RequiresProject` y `RequiresMember` |
| HU-14 | `src/web/board_handlers.go:41` | `POST /tablero/estado` (libre) | `RequiresProject` |
| HU-24 | `src/web/effort_handlers.go:28` | `POST /esfuerzo` (libre) | `RequiresProject` y `RequiresMember` |
| HU-21 | `src/web/projects_handlers.go:30` | `POST /proyectos` (libre: crear proyecto) | sin requisitos |
| HU-22 | `src/web/backlog_handlers.go:33` | `POST /backlog` (libre) | — |

**Rutas que todavía no están en `main` (T001)**

| Historia | Ruta y handler | Documento |
|---|---|---|
| HU-11 | `POST /proyectos/{id}/modificar` (`update`), en el mapa de `RegisterProjects` | `specs/011-project-update/plan.md:107,128` |
| HU-15 | `POST /cierre` con `sprint_id`, en `src/web/close_handlers.go`; sección `{Name: "Cierre de Sprint", Path: "/cierre", RequiresProject: true}` | `specs/015-sprint-close/plan.md:108,123-127` |
| HU-05b | `POST /planning-poker/cerrar` (`close`), en el `Routes` de `RegisterPlanningPoker` | `specs/005b-poker-reveal/plan.md:11,100,109` |
| HU-18 | `POST /planning-poker/nueva-ronda` (`nextRound`) y `POST /planning-poker/estimacion` (`agree`), en el mismo `Routes` | `specs/018-poker-consensus/plan.md:11,101,109-110` |

**HU-27 (T001: HU-27)**, de `specs/027-member-password/`:

- `POST /contexto/integrante` exige `contrasena`;
- `RegisterContextWithLockout`;
- `ContextStore.MemberCredential`;
- la migración `<NNN>_add_member_password` (`members.password_hash`);
- el comando `src/cmd/set-password`;
- los ayudantes de prueba: `setTestPassword(t, db, memberID)`, `memberLoginValues(id)` y `testMemberPassword` en
  `src/web`; `configurarClave(t, db, id, clave)`, `formIdentificacion(id, clave)` y `claveDePrueba` en `test/`;
- `layout.html` con `<div class="app-context">` y el botón "Salir".

**HU-11 y el paquete `src/project`**: HU-11 agrega `project.ErrProjectNotFound` en `src/project/update_errors.go`
(`specs/011-project-update/data-model.md:58`) y otros archivos `update_*.go`. Definir otro `ErrProjectNotFound` en
`src/project` chocaría al integrar.

**Base de datos**:

- `project_members (project_id, member_id)` con PK compuesta y `ON DELETE CASCADE` (`001_create_projects.up.sql`);
- HU-01 inserta solo esas dos columnas (`src/project/postgres_store.go:47`), así que el valor por defecto
  'Integrante' cubre los proyectos nuevos;
- HU-11 borra las bajas e inserta las altas (`specs/011-project-update/data-model.md:120,139`), así que conserva el
  rol de quienes siguen;
- migraciones en `main`: 001 a 004, 007, 008 y 011. HU-27 toma la siguiente libre; HU-28, la que siga.

**Pruebas existentes que hacen POST a rutas protegidas** (hoy en `main`):

| Archivo | Historia | Cómo se identifica hoy |
|---|---|---|
| `src/web/sprints_handlers_test.go` | HU-23 | doble `sprintHandlerContextStore` (sin integrante; la sección no lo pide) y, en la línea 950, `PostgreSQLContextStore` real |
| `test/web_sprints_bdd_test.go` | HU-23 | doble `sprintBDDContextStore` (línea 131) y `PostgreSQLContextStore` real (línea 547) |
| `src/web/poker_handlers_test.go` | HU-05a | `POST /contexto/integrante` (con contraseña después de HU-27) |
| `src/web/poker_view_test.go` | HU-05a | igual |
| `test/web_poker_bdd_test.go` | HU-05a | igual |

Se suman las pruebas de HU-11, HU-15, HU-05b y HU-18 cuando entren (T001 las cuenta).
`src/web/return_to_test.go`, `sections_test.go` y `test/web_base_bdd_test.go` nombran "/sprints" solo como
dirección, sin POST.

## Decisiones

1. **Paquete nuevo `src/role`**, en lugar de un archivo en `src/project`.
   - Contenido: `Role`, `RoleScrumMaster`, `RoleMember`, `ParseRole`, `CanManage`, el puerto `Store` y
     `NewPostgreSQLStore`, con el mismo patrón que `poker.NewPostgreSQLStore`.
   - Motivo: HU-11 agrega `ErrProjectNotFound` y varios archivos a `src/project`; un paquete aparte evita choques de
     nombres y de merge.
   - Costo: `role.Role` y `role.RoleScrumMaster` repiten el nombre del paquete. Se aceptan para conservar los nombres
     de la decisión del equipo.

2. **`ParseRole` exacto, con recorte de bordes.**
   - Acepta solo "Scrum Master" o "Integrante", después de `strings.TrimSpace`; cualquier otro texto (incluido "scrum
     master") da `ErrInvalidRole`.
   - Motivo: el `CHECK` de la base usa esos dos textos exactos y la estructura común los muestra tal cual.
   - Alternativa descartada: aceptar mayúsculas y minúsculas distintas, porque abre la puerta a escribir variantes
     y necesita normalizar.

3. **El rol viaja en la consulta del contexto** (decisión del equipo).

   ```sql
   SELECT p.name, m.display_name, pm.role,
          EXISTS (SELECT 1 FROM project_members sm
                  WHERE sm.project_id = p.id AND sm.role = 'Scrum Master')
   FROM projects p
   LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.member_id = $2
   LEFT JOIN members m ON m.id = pm.member_id
   WHERE p.id = $1
   ```

   - El `EXISTS` usa la PK, que empieza por `project_id`; la prueba de 50 ms de HU-20b lo controla (SC-006).
   - `ResolvedContext` suma `MemberRole role.Role` (vacío si no pertenece) y `HasScrumMaster bool`; `ActiveContext`
     suma los mismos dos campos.
   - Los dobles existentes devuelven los campos en cero: siguen compilando, pero sus POST a rutas protegidas quedan
     rechazados hasta el Bloque E.

4. **`RequireScrumMaster` como método de `Server`.**
   - Firma: `func (s *Server) RequireScrumMaster(h http.Handler) http.Handler`.
   - Motivo: el 403 necesita el `Renderer` para la página común, y las funciones `Register…` ya tienen el servidor.
     Se usa así: `"POST /sprints": server.RequireScrumMaster(http.HandlerFunc(h.create))`.
   - Alternativa descartada: una función libre que reciba el `Renderer`; cambia más líneas en cada `Register…`.

5. **HU-11, solo el proyecto activo** (respuesta de `/speckit-clarify`).
   - Firma: `func (s *Server) RequireScrumMasterOfPathProject(h http.Handler) http.Handler`.
   - Hace los mismos pasos 1 y 2 y compara `parseID(r.PathValue("id"))` con `ProjectID`. Si es distinto, inválido o
     inexistente, responde 403 con `MsgScrumMasterOtherProject`, sin consultar el rol en otro proyecto. Si coincide,
     sigue con los pasos 3 a 5.
   - El nombre del valor de ruta (`id`) se confirma en T001 con el código de HU-11.

6. **Orden del control** (regla 1 de la spec). Sin proyecto, `MsgProjectRequired` y 303 a elegir proyecto. Sin
   integrante, `MsgMemberRequired` y 303 a `pathSelectMember?volver_a=/<primer segmento>`.
   - Motivo: el envoltorio no conoce la `Section`. El primer segmento de la dirección es justo el `Path` de la
     sección en las 9 rutas (`/sprints`, `/cierre`, `/planning-poker` y `/proyectos`), y `SanitizeReturnTo` lo
     acepta.

7. **Textos y título del 403** en `src/web/scrum_master.go`:
   - `TitleScrumMasterOnly = "Acción reservada al Scrum Master"`;
   - `MsgScrumMasterOnly`;
   - `MsgNoScrumMaster`.

   Se usa `RenderError(w, r, http.StatusForbidden, …)`. No se reutiliza `TitleForbidden` ("Solicitud rechazada"),
   que habla de envíos de otro origen.

8. **"Nombre (Rol)" en `layout.html`**: `{{.MemberName}} ({{.MemberRole}})` dentro del `<strong>` que ya existe,
   solo si hay integrante. El texto queda "Integrante: Ana (Scrum Master)" o "Integrante: Beto (Integrante)".

9. **Comando `set-role` con el mismo diseño que `set-password` (T001: HU-27).**
   - `run(ctx, args, stdout, getenv) int` lee las opciones, `ParseRole` y `DATABASE_URL` (con `web.LoadConfig`);
     abre la base y delega en `assignRole(ctx, store role.Store, projectID, memberID, r role.Role, stdout) int`.
   - Se agrega `ctx` a la firma de la decisión del equipo, igual que en `set-password` y `src/cmd/server`.
   - El rol se valida antes de conectarse, y el proyecto antes que el integrante. Así el escenario 11 informa un solo
     motivo por vez, en un orden fijo.
   - Códigos de salida: 0, 1 (rechazo o falla) y 2 (opciones faltantes o números inválidos).
   - Mensajes: "<Nombre> ahora es <Rol> de «<Proyecto>»", "El rol no es válido", "El proyecto no existe", "El
     integrante no pertenece al proyecto" y "No se pudo acceder a la base de datos".

10. **Puerto `role.Store`.**
    - `ProjectName(ctx, projectID) (string, error)`: devuelve `ErrProjectNotFound` si no existe.
    - `SetMemberRole(ctx, projectID, memberID, r) (memberName string, err error)`: un `UPDATE … FROM members …
      RETURNING display_name`; sin filas, `ErrMemberNotInProject`.
    - Los errores son de `src/role`, sin choque con HU-11.

11. **Pruebas existentes: un ayudante común por paquete, más los dobles.**
    - `src/web`: `setTestScrumMaster(t, db, projectID, memberID)` llama a `setTestPassword` (T001: HU-27) y pone
      `role = 'Scrum Master'`.
    - `test/`: `configurarScrumMaster(t, db, proyecto, integrante)` hace lo mismo con `configurarClave` y
      `claveDePrueba`.
    - Los dobles de HU-23 (`sprintHandlerContextStore` y `sprintBDDContextStore`) reciben un integrante de prueba que
      devuelven como Scrum Master. El cliente lleva su cookie firmada con `SignedContextValue(CookieMember, id)`: con
      un doble no hay contraseña que verificar.

12. **Control por tabla de rutas.**
    - `TestScrumMasterRoutes_Protegidas` registra las secciones reales con dobles de las acciones que cuentan
      llamadas.
    - Recorre la tabla de las rutas protegidas presentes y comprueba el 403 para Beto y para «Beta» (sin Scrum
      Master), con 0 llamadas a la acción.
    - `TestScrumMasterRoutes_AccionesLibres` hace lo mismo con las rutas libres y comprueba que sí llegan.

13. **BDD con el cableado real.**
    - `test/web_member_roles_bdd_test.go` arma el servidor como `main.go`, con los `Register…` reales y los
      adaptadores PostgreSQL de cada historia, para que "el Sprint se cierra como indica HU-15" y "el proyecto no
      cambia" se comprueben en la base.
    - Los nombres de las funciones de registro de HU-15 y HU-11 se confirman en T001.

14. **Número de la migración**: la mayor en `main` + 1, después de la de HU-27, sin rellenar huecos (golang-migrate
    solo aplica versiones mayores a la actual). T001 la fija y T033 la reconfirma.

## Datos a reconfirmar en T001 (el código pudo cambiar)

- Que HU-27 esté en la rama base (`git grep -n "RegisterContextWithLockout\|password_hash"`); si no está, **detener
  todo y avisar**.
- Las rutas reales de las 9 acciones: `git grep -n '"POST ' src/web`, y los nombres de los handlers, de las funciones
  de registro de HU-15 y HU-11, y del valor de ruta `{id}` de HU-11.
- Las pruebas que hacen POST a esas rutas, incluidas las de HU-11, HU-15, HU-05b y HU-18.
- Los dobles que implementan `ContextStore` y los nombres reales de los ayudantes de HU-27.
- La mayor migración en `main`.

## Preguntas abiertas

Ninguna.
