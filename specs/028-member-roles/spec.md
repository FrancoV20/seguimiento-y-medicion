# Especificación: HU-28 – Roles del Proyecto (mínimos)

**Rama**: `HU-28`
**Creada**: 2026-10-28
**Estado**: Propuesta (pendiente de aceptación del equipo)
**Constitución**: v1.1.0
**Prioridad**: Baja – Could Have (trabajo futuro si no llega; se integra después de HU-27) | **Valor**: 55 |
**Estimación**: 5 Story Points (sugerida)
**Depende de**:

- en `main`: HU-20b, HU-23 y HU-05a;
- todavía no en `main`: HU-27, HU-11, HU-15, HU-05b y HU-18.

Los nombres de esas historias salen de sus documentos y se marcan "(T001: HU-27)" o "(T001)".

## Historia de usuario

Como Scrum Master, quiero que solo yo pueda planificar y cerrar los Sprints, modificar el proyecto y conducir las
rondas de Planning Poker, para que las decisiones de facilitación y planificación no las tome cualquier integrante
por error.

## Clarifications

### Session 2026-10-28

Decisiones del equipo (dadas al pedir la especificación; no se volvieron a preguntar):

- Q: ¿Qué roles hay y dónde se guardan? → A: "Scrum Master" e "Integrante", por proyecto, en `project_members.role`.
  Una migración nueva:
  `ALTER TABLE project_members ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'Integrante' CHECK (role IN ('Scrum Master', 'Integrante'))`
  (down: `DROP COLUMN`). El número es el siguiente libre en `main` al implementar; lo confirman T001 y T033, la tarea
  previa al PR. Puede haber más de un Scrum Master por proyecto.
- Q: ¿Dónde vive la regla? → A: En un paquete chico de dominio, con `Role`, `RoleScrumMaster`, `RoleMember`,
  `ParseRole` y `CanManage(role)`.
- Q: ¿Cómo se lee el rol? → A: Se amplía el contexto de HU-20b: `ActiveContext` lleva el rol del integrante
  identificado y si el proyecto tiene algún Scrum Master. Los resuelve el middleware en la misma consulta que hoy
  valida la pertenencia. Adaptador PostgreSQL con pruebas de integración.
- Q: ¿Cómo se controla? → A: Con un envoltorio de handler `RequireScrumMaster(h)` en `src/web`:
  - integrante identificado con rol "Scrum Master": llama a `h`;
  - sin integrante identificado: la redirección de HU-20b;
  - integrante sin el rol: 403, "Esta acción la puede hacer solo el Scrum Master del proyecto";
  - proyecto sin ningún Scrum Master: 403, "Este proyecto no tiene Scrum Master asignado. Pedí que se configure con el
    comando de roles".

  La página 403 tiene la estructura común (como la de HU-20b para envíos de otro origen, con su propio texto). Se
  aplica **solo** a estas rutas POST, envueltas donde se registran (`Routes` de cada `Register…`) sin cambiar el
  interior de los handlers:
  - modificar proyecto (HU-11);
  - crear Sprint y agregar o quitar historias (HU-23);
  - cerrar Sprint (HU-15);
  - abrir y cerrar ronda (HU-05a y HU-05b);
  - nueva ronda y estimación acordada (HU-18).

  T001 lista las rutas reales con `git grep`.
- Q: ¿Qué se ve en las pantallas? → A: Los botones siguen visibles para todos; deshabilitarlos es una mejora futura.
  La estructura común muestra "Nombre (Rol)" del integrante identificado.
- Q: ¿Cómo se asigna un rol? → A: Con el comando `src/cmd/set-role`, con `-proyecto`, `-integrante` y `-rol`. Mensajes:
  - "Beto ahora es Scrum Master de «Alfa»";
  - "El rol no es válido";
  - "El integrante no pertenece al proyecto";
  - "El proyecto no existe".

  La lógica va en `run(args, stdout, getenv)`, con prueba de integración.
- Q: ¿Qué pasa con las pruebas existentes? → A: Las de las acciones de la lista se identifican como Scrum Master con
  un ayudante común, que configura el rol y la contraseña de la fixture. Sus expectativas no cambian, salvo la
  identificación.
- Q: ¿Qué queda fuera? → A: Product Owner como rol, pantalla de roles, deshabilitar botones y permisos por sección.

Preguntas de `/speckit-clarify`:

- Q: Si el Scrum Master de «Alfa», con «Alfa» activo, intenta guardar cambios en otro proyecto
  (`POST /proyectos/{id}/modificar` con el `{id}` de «Beta»), ¿qué pasa? → A: Solo se puede modificar el proyecto
  activo. Si el `{id}` no es el proyecto activo (distinto, inválido o inexistente), 403 con un mensaje propio:
  "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master", sin consultar el rol en otro proyecto. Si el `{id}` es el activo, se aplican las
  reglas normales. (Corrección del equipo, 28/10: el mensaje es propio, no el genérico; se agrega el escenario 13.)

## Sección SDD

### Intención

Que las acciones de planificación y facilitación (planificar y cerrar Sprints, modificar el proyecto y conducir el
Planning Poker) solo las pueda hacer quien tiene el rol de Scrum Master en el proyecto activo. HU-27 asegura quién es
el integrante; HU-28 agrega qué puede hacer. Votar, mover historias en el Tablero y registrar esfuerzo siguen abiertos
a cualquier integrante identificado.

### Alcance

- Rol por integrante y proyecto ("Scrum Master" o "Integrante"); todos empiezan como "Integrante".
- Lectura del rol en cada solicitud, junto con la validación del contexto de HU-20b.
- "Nombre (Rol)" en la estructura común.
- Control en el servidor de las 8 rutas POST de la lista, con dos rechazos 403 y la redirección de HU-20b.
- Comando `go run ./src/cmd/set-role -proyecto P -integrante N -rol "Scrum Master"`.
- Migración que agrega `project_members.role` (up y down).
- Actualización de las pruebas existentes de esas acciones.

### Fuera de alcance

- Product Owner como rol.
- Pantalla para ver o asignar roles.
- Ocultar o deshabilitar los botones según el rol.
- Permisos por sección completa o sobre pantallas `GET`.
- Impedir que un proyecto quede sin Scrum Master (por ejemplo, si HU-11 quita al único).
- Restringir crear proyectos (HU-21), crear historias del Backlog (HU-22), votar, cambiar estados en el Tablero o
  registrar esfuerzo.

### Entradas

- El integrante identificado y el proyecto activo (cookies firmadas de HU-26, identificación de HU-27).
- Las 8 solicitudes POST de la lista.
- Comando: opciones `-proyecto P`, `-integrante N` y `-rol R`, y la variable `DATABASE_URL`.

### Salidas

- "Nombre (Rol)" en la estructura común.
- Páginas 403 con los dos textos, o la acción normal de cada historia.
- Columna `project_members.role`.
- Salida del comando y su código de salida.

### Rutas protegidas

| Historia | Ruta | Estado en `main` |
|---|---|---|
| HU-11 | `POST /proyectos/{id}/modificar` | no está (T001) |
| HU-23 | `POST /sprints` | está |
| HU-23 | `POST /sprints/historias` | está |
| HU-23 | `POST /sprints/historias/quitar` | está |
| HU-15 | `POST /cierre` | no está (T001) |
| HU-05a | `POST /planning-poker/abrir` | está |
| HU-05b | `POST /planning-poker/cerrar` | no está (T001) |
| HU-18 | `POST /planning-poker/nueva-ronda` | no está (T001) |
| HU-18 | `POST /planning-poker/estimacion` | no está (T001) |

Son 9 patrones para las 8 acciones de la nota 2 ("agregar o quitar historias" son dos rutas). Siguen libres para
cualquier integrante:

- `POST /planning-poker/votar`, `POST /tablero/estado` y `POST /esfuerzo`;
- `POST /backlog` y `POST /proyectos`;
- las rutas de `/contexto/…`;
- todas las `GET`.

### Reglas

1. Orden del control, después de `RequireContext` de la sección:
   1. sin proyecto activo → elegir proyecto (HU-20b);
   2. sin integrante identificado → "Identificarme" (HU-20b, con contraseña por HU-27);
   3. rol "Scrum Master" → la acción;
   4. el proyecto no tiene ningún Scrum Master → 403 "Este proyecto no tiene Scrum Master asignado…";
   5. en otro caso → 403 "Esta acción la puede hacer solo el Scrum Master del proyecto".

   En `POST /proyectos/{id}/modificar`, además, el `{id}` debe ser el proyecto activo; si no lo es, 403
   "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master", aunque el integrante sea Scrum Master del proyecto
   activo y sin consultar el rol en otro proyecto. Esa comprobación va después del paso 2 y antes del 3.
2. El rechazo ocurre antes de llamar al handler de la historia: no se valida el formulario, no se escribe nada y no se
   cuenta nada.
3. El rol se lee en cada solicitud: un cambio hecho con el comando vale desde la siguiente solicitud, sin volver a
   identificarse.
4. La redirección por falta de identificación vuelve a la sección de la acción (por ejemplo, `/cierre`), como hace
   HU-20b con los envíos; los datos del formulario no se conservan.
5. El texto del rol que se muestra es el guardado: "Scrum Master" o "Integrante".

### Restricciones

- Go 1.27.1, PostgreSQL y la biblioteca estándar; `go.mod` no cambia.
- Renderizado en el servidor, sin JavaScript.
- Los handlers de las otras historias no cambian por dentro; solo se envuelven sus rutas al registrarlas.
- Se integra después de HU-27, al final de todo.

### Condiciones de error

| Situación | Respuesta | ¿Cambia algo? |
|---|---|---|
| Sin proyecto activo | 303 a elegir proyecto (HU-20b) | No |
| Sin integrante identificado | 303 a "Identificarme" con `volver_a` a la sección (HU-20b) | No |
| Integrante con rol "Integrante" y el proyecto tiene Scrum Master | 403, "Esta acción la puede hacer solo el Scrum Master del proyecto" | No |
| Modificar un proyecto (`{id}`) que no es el activo (distinto, inválido o inexistente) | 403, "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master" | No |
| El proyecto no tiene ningún Scrum Master | 403, "Este proyecto no tiene Scrum Master asignado. Pedí que se configure con el comando de roles" | No |
| Falla la lectura del contexto (rol incluido) | 503 o 500 de HU-20a, sin detalles | No |
| Envío de otro origen | 403 de HU-20b (antes que todo) | No |
| Scrum Master | lo que responda la historia | según la historia |

## Datos de ejemplo

Del Issue, sin cambios:

- Proyecto «Alfa»: Ana es "Scrum Master"; Beto y Caro son "Integrante". Todos tienen contraseña (HU-27).
- Proyecto «Beta»: Dani es "Integrante" y no tiene Scrum Master asignado.
- Sprint 2 de «Alfa» en curso; historia «Login» con su ronda 1 abierta.

Agregado para las pruebas (el Issue no lo da):

- Contraseñas: las de HU-27 ("ana-clave-2026", "beto-clave-2026", "dani-clave-2026") y "caro-clave-2026" para Caro,
  que en HU-28 tiene contraseña.
- «Login» pertenece al Backlog de «Alfa» y está en el Sprint 2. Para crear Sprints y agregar historias hay otra
  historia del Backlog, «Perfil», sin Sprint.
- Para el escenario 9, «Beta» tiene una historia en su Backlog, «Reportes», para que crear un Sprint tenga datos
  válidos.
- «Alfa» tiene un Sprint 1 ya finalizado, para que el Sprint en curso sea el número 2.

## Escenarios BDD

Texto del Issue HU-28.

**Escenario 1: Rol visible**
Dado que me identifiqué como Ana en «Alfa»
Cuando navego por la aplicación
Entonces veo "Ana (Scrum Master)" en la estructura común

**Escenario 2: El Scrum Master cierra el Sprint**
Dado que estoy identificado como Ana
Cuando cierro el Sprint 2
Entonces el Sprint se cierra como indica HU-15

**Escenario 3: Un integrante no puede cerrar el Sprint**
Dado que estoy identificado como Beto
Cuando intento cerrar el Sprint 2
Entonces veo "Esta acción la puede hacer solo el Scrum Master del proyecto" y el Sprint sigue en curso

**Escenario 4: Un integrante no puede crear Sprints ni asignar historias**
Dado que estoy identificado como Beto
Cuando intento crear un Sprint o agregar o quitar historias del Sprint en curso
Entonces veo "Esta acción la puede hacer solo el Scrum Master del proyecto" y no cambia nada

**Escenario 5: Un integrante no puede modificar el proyecto**
Dado que estoy identificado como Caro
Cuando intento guardar cambios en «Alfa»
Entonces veo "Esta acción la puede hacer solo el Scrum Master del proyecto" y el proyecto no cambia

**Escenario 6: Un integrante no puede conducir el Planning Poker**
Dado que estoy identificado como Beto
Cuando intento abrir o cerrar una ronda, abrir una nueva ronda o registrar la estimación acordada
Entonces veo "Esta acción la puede hacer solo el Scrum Master del proyecto" y no cambia nada

**Escenario 7: Un integrante sí puede votar y trabajar**
Dado que estoy identificado como Beto
Cuando voto en la ronda de «Login», cambio el estado de una historia en el Tablero o registro esfuerzo
Entonces la acción se hace normalmente

**Escenario 8: Acción de Scrum Master sin identificarse**
Dado que no me identifiqué
Cuando intento cerrar el Sprint 2
Entonces se me pide identificarme y no cambia nada

**Escenario 9: Proyecto sin Scrum Master**
Dado que estoy identificado como Dani en «Beta»
Cuando intento crear un Sprint
Entonces veo "Este proyecto no tiene Scrum Master asignado. Pedí que se configure con el comando de roles" y no cambia
nada

**Escenario 10: Asignar el rol**
Dado que Beto es "Integrante" de «Alfa»
Cuando se ejecuta el comando de roles para darle a Beto el rol "Scrum Master" en «Alfa»
Entonces el comando muestra "Beto ahora es Scrum Master de «Alfa»" y Beto puede cerrar el Sprint

**Escenario 11: Rol inválido al asignar**
Dado el comando de roles
Cuando se indica un rol que no existe, un integrante que no es del proyecto o un proyecto inexistente
Entonces el comando informa el motivo y no cambia nada

**Escenario 12: Error al verificar el rol**
Dado un fallo al leer el rol del integrante
Cuando intento una acción de Scrum Master
Entonces veo la página de error de la aplicación y no cambia nada

**Escenario 13: Modificar un proyecto que no es el activo**
Dado que estoy identificado como Ana, Scrum Master de «Alfa», con «Alfa» activo
Cuando intento guardar cambios en «Beta»
Entonces veo "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master" y «Beta» no cambia

## Casos límite

- **Más de un Scrum Master**: los dos pueden hacer todas las acciones.
- **Modificar otro proyecto**: el Scrum Master de «Alfa» con «Alfa» activo no puede guardar cambios en «Beta», aunque
  también sea Scrum Master de «Beta»: primero tiene que elegir «Beta» e identificarse ahí. Un `{id}` inválido o
  inexistente también es "otro proyecto" y da el mismo 403 con el mensaje propio (escenario 13), antes de llegar a
  HU-11.
- **Cambio de rol con la identificación abierta**: vale desde la siguiente solicitud (regla 3), en los dos sentidos.
- **Proyecto recién creado (HU-21) o existente al migrar**: todos sus integrantes quedan como "Integrante"; nadie
  puede hacer las acciones de la lista hasta asignar un Scrum Master con el comando (escenario 9).
- **HU-11 quita al único Scrum Master**: el proyecto queda sin Scrum Master; se resuelve con el comando (fuera de
  alcance impedirlo).
- **Integrante identificado que deja de pertenecer al proyecto**: HU-20b limpia la identificación, y la acción pide
  identificarse.
- **Acción de Scrum Master desde una sección sin `RequiresMember`** (Sprints, Cierre, Proyectos): el envoltorio pide
  identificarse aunque la sección no lo exija para mirar.
- **Planning Poker** (`RequiresMember: true`): sin integrante, la redirección la hace `RequireContext` antes del
  envoltorio; el resultado es el mismo.
- **Envío de otro origen a una ruta de Scrum Master**: 403 de HU-20b, antes que el control de rol.
- **`GET` de las mismas direcciones** (`/sprints`, `/cierre`, `/planning-poker`, el formulario de modificar proyecto):
  sin cambios; los botones siguen visibles.
- **Comando con `-rol` en otra capitalización o con otro texto** ("scrum master", "SM", "Product Owner"): "El rol no es
  válido". Se recortan los espacios de los bordes.
- **Comando con un integrante que existe pero no es del proyecto, o que no existe**: "El integrante no pertenece al
  proyecto".
- **Comando que asigna el rol que ya tenía**: lo informa igual ("… ahora es …") y termina bien.
- **Comando que quita el rol al último Scrum Master**: se permite; el proyecto queda sin Scrum Master.
- **Comando sin alguna opción o con un número inválido**: muestra cómo se usa y termina con código 2, sin conectarse.
- **Comando sin `DATABASE_URL`**: el mismo mensaje que el servidor, código 1.

## Requisitos funcionales

- **FR-001**: Cada integrante de un proyecto MUST tener un rol en ese proyecto: "Scrum Master" o "Integrante"; por
  defecto, "Integrante". Un proyecto MAY tener más de un Scrum Master.
- **FR-002**: La estructura común MUST mostrar "<Nombre> (<Rol>)" del integrante identificado.
- **FR-003**: Las rutas de la tabla "Rutas protegidas" MUST exigir un integrante identificado con rol "Scrum Master"
  en el proyecto activo. Las demás rutas MUST NOT cambiar.
- **FR-004**: Un Scrum Master identificado MUST poder hacer esas acciones con el mismo resultado que hoy define cada
  historia.
- **FR-005**: Un integrante con rol "Integrante", en un proyecto que tiene Scrum Master, MUST recibir 403 con "Esta
  acción la puede hacer solo el Scrum Master del proyecto", y la acción MUST NOT ejecutarse.
- **FR-006**: Si el proyecto activo no tiene ningún Scrum Master, MUST responderse 403 con "Este proyecto no tiene
  Scrum Master asignado. Pedí que se configure con el comando de roles", y la acción MUST NOT ejecutarse.
- **FR-007**: Sin integrante identificado, la acción MUST redirigir a "Identificarme" como HU-20b (con `volver_a` a la
  sección); sin proyecto activo, a elegir proyecto. La acción MUST NOT ejecutarse.
- **FR-008**: El control MUST ocurrir antes de llamar al handler de la historia, aplicado al registrar la ruta, sin
  cambiar el interior de los handlers de las otras historias.
- **FR-009**: Votar, cambiar el estado en el Tablero y registrar esfuerzo MUST seguir funcionando para cualquier
  integrante identificado.
- **FR-010**: El rol MUST leerse de los datos actuales en cada solicitud, en la misma consulta que valida el contexto
  de HU-20b.
- **FR-011**: Si falla la lectura del rol, MUST mostrarse la página de error de HU-20a (503 o 500) sin detalles
  internos, y la acción MUST NOT ejecutarse.
- **FR-012**: Las páginas 403 MUST usar la estructura común, con un enlace al inicio y sin detalles internos. Los
  botones de las pantallas MUST seguir visibles.
- **FR-013**: El comando `set-role` con `-proyecto`, `-integrante`, `-rol` y `DATABASE_URL` MUST asignar el rol y
  mostrar "<Nombre> ahora es <Rol> de «<Proyecto>»".
- **FR-014**: Ante un rol inválido ("El rol no es válido"), un integrante que no es del proyecto ("El integrante no
  pertenece al proyecto") o un proyecto inexistente ("El proyecto no existe"), el comando MUST informar el motivo,
  MUST NOT cambiar nada y MUST terminar con un código distinto de 0.
- **FR-015**: La migración MUST agregar `project_members.role` (texto de hasta 20 caracteres, no nulo, por defecto
  'Integrante', solo 'Scrum Master' o 'Integrante') y su reversión MUST quitarla.
- **FR-016**: `POST /proyectos/{id}/modificar` MUST rechazarse con 403 y el texto
  "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master" cuando el `{id}` no es el proyecto activo (distinto,
  inválido o inexistente), sin consultar el rol en otro proyecto y sin llamar al handler de HU-11. Si el `{id}` es el
  proyecto activo, MUST aplicarse la regla 1 sin cambios.
- **FR-017**: Las pruebas existentes de las acciones de la lista MUST seguir en verde identificándose como Scrum Master
  con un ayudante común; sus demás expectativas no cambian.

## Entidades clave

- **Rol**: "Scrum Master" o "Integrante", de un integrante en un proyecto (`project_members.role`).
- **Contexto activo**: suma el rol del integrante identificado y si el proyecto tiene algún Scrum Master.

## Criterios de éxito

- **SC-001**: Cada uno de los 13 escenarios tiene al menos una prueba que falló antes del código (evidencia RED) y
  pasa después.
- **SC-002**: El 100% de las rutas protegidas presentes en `main` rechaza a un "Integrante" y a un proyecto sin Scrum
  Master sin cambiar datos, en una prueba por tabla sobre todas ellas.
- **SC-003**: El 100% de las acciones libres (votar, Tablero, esfuerzo) funciona para un "Integrante".
- **SC-004**: Toda la suite (`go test -count=1 -p 1 ./...`) queda en verde; en las pruebas existentes solo cambia la
  identificación.
- **SC-005**: La migración se aplica y se revierte sin errores sobre una base descartable.
- **SC-006**: La validación del contexto con el rol sigue por debajo de 50 ms por solicitud (SC-012 de HU-20b).

## Supuestos y decisiones

- La identificación con contraseña de HU-27 (`POST /contexto/integrante` con `contrasena`), su comando `set-password` y
  su migración `members.password_hash` se toman de `specs/027-member-password/` (T001: HU-27).
- Las rutas de HU-11, HU-15, HU-05b y HU-18 se toman de sus planes (T001). Si alguna no está en `main` al implementar,
  se protege en la rama de esa historia (aviso de T003).
- Al migrar, todos los integrantes existentes quedan como "Integrante". Para la demo hay que asignar al menos un Scrum
  Master por proyecto con el comando.
- El Scrum Master conoce los números del proyecto y del integrante (por ejemplo, con SQL); no hay pantalla para verlos.
- "Planificación de proyecto.pdf" no está en el repositorio; si dice otra cosa, prevalece y se corrige esta
  especificación antes de T002.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba BDD (`test/`) | Pruebas unitarias, de handler e integración | Tareas |
|---|---|---|---|---|---|
| 1 Rol visible | FR-002, FR-010 | SC-001 | `TestHU28_RolVisible` | `TestContexto_RolEnLaEstructuraComun`, `TestPostgreSQLContextStore_ResolveContextConRol` | T013, T014, T020 |
| 2 Scrum Master cierra | FR-003, FR-004 | SC-001 | `TestHU28_ScrumMasterCierraSprint` | `TestRequireScrumMaster_ScrumMasterPasa`, `TestScrumMasterRoutes_Protegidas` | T018, T019, T020 |
| 3 Integrante no cierra | FR-005, FR-008 | SC-001, SC-002 | `TestHU28_IntegranteNoCierraSprint` | `TestRequireScrumMaster_IntegranteRechazado`, `TestScrumMasterRoutes_Protegidas` | T018, T019, T020 |
| 4 Integrante no planifica | FR-003, FR-005 | SC-001, SC-002 | `TestHU28_IntegranteNoPlanificaSprint` | `TestScrumMasterRoutes_Protegidas` | T019, T020 |
| 5 Integrante no modifica el proyecto | FR-005 | SC-001, SC-002 | `TestHU28_IntegranteNoModificaProyecto` | `TestScrumMasterRoutes_Protegidas` | T019, T020 |
| 6 Integrante no conduce el Poker | FR-003, FR-005 | SC-001, SC-002 | `TestHU28_IntegranteNoConducePoker` | `TestScrumMasterRoutes_Protegidas` | T019, T020 |
| 7 Integrante vota y trabaja | FR-009 | SC-001, SC-003 | `TestHU28_IntegranteVotaYTrabaja` | `TestScrumMasterRoutes_AccionesLibres` | T019, T020 |
| 8 Sin identificarse | FR-007 | SC-001 | `TestHU28_SinIdentificarse` | `TestRequireScrumMaster_SinIdentificacion`, `TestRequireScrumMaster_SinProyecto` | T018, T020 |
| 9 Proyecto sin Scrum Master | FR-006 | SC-001, SC-002 | `TestHU28_ProyectoSinScrumMaster` | `TestRequireScrumMaster_SinScrumMaster`, `TestScrumMasterRoutes_Protegidas`, `TestPostgreSQLContextStore_ResolveContextConRol` | T013, T018, T019, T020 |
| 10 Asignar el rol | FR-010, FR-013 | SC-001 | `TestHU28_AsignarRol` | `TestAssignRole_AsignaScrumMaster`, `TestRun_AsignaRol`, `TestPostgreSQLStore` | T007, T020, T025 |
| 11 Rol inválido al asignar | FR-014 | SC-001 | — | `TestParseRole`, `TestCanManage`, `TestAssignRole_Rechazos`, `TestRun_UsoYEntorno`, `TestPostgreSQLStore` | T005, T007, T025 |
| 12 Error al verificar | FR-011, FR-012 | SC-001 | `TestHU28_ErrorAlVerificarRol` | — (la página 503 sin detalles es la de HU-20b, ya cubierta por `TestContexto_ErroresNoExponenDetallesInternos`) | T020 |
| 13 Modificar otro proyecto | FR-016 | SC-001 | `TestHU28_ModificarOtroProyecto` | `TestRequireScrumMasterOfPathProject_OtroProyecto` | T018, T020 |
| Migración | FR-001, FR-015 | SC-005 | — | `TestMigracion_ProjectMemberRole_UpYDown` | T006 |
| Rendimiento del contexto | FR-010 | SC-006 | — | `TestPostgreSQLContextStore_ResolveContext_MenosDe50ms` (HU-20b, sin cambios) | T013, T017 |
| Regresión de pruebas existentes | FR-017 | SC-004 | las que liste T001 | las que liste T001 | T029–T032 |
