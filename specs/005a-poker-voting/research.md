# Investigación: HU-05a – Emisión de voto oculto en Planning Poker

## Resultado de la preparación (T001–T004, 28/10/2026)

- **T001**: `HU-05a` está en `origin/main` (`d234b5a`), sin commits propios: no hizo falta adelantarla.
- **T002**: `git fetch --all`; en `origin/main` y en todas las ramas remotas la última migración es `008`. Nadie
  usa `009`, `010` ni `011`: HU-05a toma `011` (009 y 010 siguen reservadas para HU-06 y HU-15).
- **T003**: `git grep -n -i poker -- src` sin coincidencias. En `origin/main` están `RequireContext`, `parseID`,
  `Renderer.PageStatus`, `Renderer.Fail`, `SetFlash`, `MsgOperationFailed`, el parcial `alert` y
  `backlog.ValidateStoryPoints(*int) error`.
- **T023**: la migración `011` se probó con `up`, `down 1` y `up` (las tablas `poker_rounds` y `poker_votes`
  aparecen, desaparecen y vuelven a aparecer) y con `go test -count=1 -p 1 ./db/...`, que aplica todas las
  migraciones sobre una base descartable.
- **T004**: línea base en verde con `go test -count=1 -p 1 ./...` (PostgreSQL 16, migraciones 001–008).

## Datos verificados en el código (27/10/2026)

- **Rama de trabajo**: la `main` local está en `dc55f92` y `origin/main` en `774fd05` (ya incluye HU-14, el
  Tablero). La rama `HU-05a` ya existe localmente y apunta a `dc55f92` (la misma `main` local, sin commits
  propios). T001 actualiza `main` y adelanta `HU-05a` a `origin/main` con
  `git merge --ff-only` (o la crea si no existiera).
- **Tipos de los identificadores** (migraciones 001–004):
  - `projects.id` y `members.id`: `SERIAL` (INTEGER). `project_members(project_id, member_id)` son INTEGER.
  - `user_stories.id`: `BIGSERIAL`; `user_stories.project_id`: BIGINT sin borrado en cascada.
  - `effort_logs.member_id` (HU-03): INTEGER `REFERENCES members(id)` sin cascada. Es el precedente que se
    sigue para las referencias a integrantes.
  - En Go todos se manejan como `int64`; `web.parseID` acepta de 1 a `MaxID` (`math.MaxInt32`).
- **Story Points**: `src/backlog/story_points.go` tiene el mapa privado `validStoryPoints` y la función
  **exportada** `backlog.ValidateStoryPoints(*int) error`, que devuelve `backlog.ErrInvalidStoryPoints`. La prueba
  de la serie la llama desde `src/poker` sin editar `src/backlog`. La columna `user_stories.story_points` tiene
  el mismo CHECK `IN (1,2,3,5,8,13,21,34,55,89)`.
- **Contratos web** (en `main`): `Server.Register(Section, Routes)`, `Section{Name, Path, RequiresProject,
  RequiresMember}`, `RequireContext` (redirige con 303 a `/contexto/proyecto` o `/contexto/integrante` con
  `volver_a`; en un POST vuelve a `Section.Path`), `ActiveContextFrom` (`ProjectID`, `ProjectName`, `MemberID`,
  `MemberName`), `SetFlash`, `RedirectSeeOther`, `Renderer.PageStatus`, `PageData{Title, Form, Data, Flash}`,
  `Renderer.Fail`, `NewFormView`, `parseID`, `MsgOperationFailed`, `CrossOriginGuard`. El parcial `alert` está
  en `templates/layout.html`.
- **Integrante identificado**: `contextMiddleware` valida en cada solicitud, con `ResolveContext`, que el
  integrante de la cookie pertenezca al proyecto activo (`project_members`); si no pertenece, lo descarta.
- **Patrón de acción POST con rechazos por código** (HU-14, `origin/main:src/web/board_handlers.go`): validar el
  envío, llamar al dominio, traducir el error a `{código, Flash}` y volver a mostrar la pantalla con
  `PageStatus`; ante un éxito, `SetFlash` + `RedirectSeeOther`. Los textos se definen en un solo archivo con
  prefijo propio (`MsgBoard…`).
- **Nombres libres**: `git grep -n -i poker -- src` no encuentra nada. No existe el paquete `src/poker` ni
  identificadores `Poker…` en `src/web`.
- **Migraciones**:
  - En `main` y `origin/main`: 001, 002, 003, 004, 007 y 008 (005 y 006 no existen; ya hay huecos).
  - `git fetch --all` (27/10): ninguna de las 18 ramas remotas (`HU-01` … `HU-26`, `HU_14`, `HU_22`,
    `Fix-HU-01`, `HU-04_Fixes`) tiene un archivo 009 o mayor. `origin/HU-06` todavía no tiene su migración.
  - `specs/006-defects` (HU-06) y `specs/015-sprint-close` (HU-15) no fijan número: dicen "último de `main` + 1".
    La reserva 009 = HU-06 y 010 = HU-15 es un acuerdo del equipo (indicado en el pedido), no figura en el repo.
  - `db/migrations_integration_test.go` aplica todos los `.up.sql` en orden sobre una base descartable;
    `web.CheckSchema` compara la versión aplicada con la mayor versión embebida.

## Decisiones

1. **Migración `011_create_poker`.**
   - Decisión: `db/migrations/011_create_poker.up.sql` y `.down.sql`.
   - Motivo: 009 y 010 están reservadas para HU-06 y HU-15 por acuerdo del equipo, y ninguna rama usa 011.
   - Alternativa descartada: 009 (el siguiente número libre en el repo), que chocaría con la reserva de HU-06.
   - **Riesgo**: `golang-migrate` registra solo la última versión aplicada. Si 011 llega a `main` antes que 009 o
     010, una base local que ya esté en 011 **no** aplica después 009 ni 010 con `migrate up` (quedan por debajo
     de la versión actual) y `CheckSchema` no lo detecta. Mitigación: T002 confirma el último número en
     `origin/main` al crear la rama y T080 lo vuelve a confirmar antes del PR. Si 009 o 010 siguen sin
     integrarse, se avisa al equipo para acordar el orden de merge o renumerar la migración propia al siguiente
     libre (nunca se editan migraciones ajenas).

2. **Paquete de dominio `src/poker` con un puerto de escritura.**
   - Decisión: reglas puras (`series.go`, `round.go`), errores (`errors.go`), puerto `Store` (`store.go`),
     `Service` (`service.go`) y adaptador `PostgreSQLStore` (`postgres_store.go`).
   - Motivo: igual que `src/defect` (HU-19) y `src/sprint` (HU-14); las reglas se prueban sin base y el adaptador
     las aplica dentro de la transacción.
   - Alternativa descartada: escribir desde `src/web` (mezcla reglas y presentación, contra el principio II).

3. **Lectura de la pantalla con un catálogo propio en `src/web`.**
   - Decisión: interfaz `PokerCatalog` y adaptador `PostgreSQLPokerCatalog` en `src/web`, con prueba de
     integración.
   - Motivo: es el patrón de HU-14 (`BoardCatalog`), HU-22 (`StoryCatalog`) y HU-24: la lectura de pantalla vive
     junto a la pantalla y el dominio solo expone escrituras.
   - Alternativa descartada: lecturas en `src/poker` (agrega al dominio tipos que solo usa la vista).

4. **El modelo de vista no tiene dónde guardar votos ajenos.**
   - Decisión: `PokerParticipantView` tiene `Name`, `HasVoted` e `IsMe`, sin campo de valor. El único valor de la
     ronda es `PokerRoundView.OwnVote` (`*int`). La consulta de participantes devuelve `EXISTS (…)` y nunca la
     columna `value`; la del voto propio filtra por el integrante identificado.
   - Motivo: el ocultamiento no depende de que la plantilla "se acuerde" de no mostrar algo: el dato no llega a la
     capa web. Una prueba de vista lo verifica con valores distintos por integrante.
   - Alternativa descartada: traer todos los votos y filtrar en la plantilla (un error de plantilla filtraría los
     valores).

5. **Abrir ronda bloquea la historia con `FOR NO KEY UPDATE`.**
   - Decisión: la transacción lee la historia del proyecto con `FOR NO KEY UPDATE`, lee sus rondas, aplica
     `CheckOpen` e inserta. El índice único parcial `(story_id) WHERE status = 'Abierta'` es la segunda defensa:
     un `23505` sobre él se traduce a `ErrRoundAlreadyOpen`.
   - Motivo: dos aperturas simultáneas de la misma historia se serializan y la segunda ve la ronda de la primera.
     `FOR NO KEY UPDATE` no bloquea los `FOR KEY SHARE` que toman las claves foráneas, así que no frena cargas
     de esfuerzo (HU-03) ni asignaciones a Sprint (HU-04) sobre esa historia.
   - Alternativa descartada: `FOR UPDATE` (también bloquea las inserciones de otras tablas que referencian la
     historia, sin beneficio); solo el índice único (la segunda apertura fallaría con un error de base en lugar
     de la regla).

6. **Votar toma la ronda con `FOR SHARE` y hace un upsert.**
   - Decisión: `SELECT status … FOR SHARE` sobre la ronda del proyecto, `CheckVote` y un único
     `INSERT … ON CONFLICT (round_id, member_id) DO UPDATE`. Un CTE que lee el voto previo indica si fue alta o
     cambio, para elegir "Voto registrado" o "Voto actualizado".
   - Motivo: `FOR SHARE` permite votos simultáneos de varios integrantes y se serializa con el `FOR UPDATE` con
     el que HU-05b cerrará la ronda: no puede quedar un voto en una ronda después de cerrarla.
   - Alternativas descartadas: `RETURNING (xmax = 0)` (depende de una columna interna de PostgreSQL); `INSERT …
     DO NOTHING` seguido de `UPDATE` (dos sentencias para lo mismo).
   - Nota: si el mismo integrante envía dos primeros votos a la vez, ambos pueden decir "Voto registrado"; queda
     un solo voto con el último valor. Es inofensivo y no se trata.

7. **El valor se valida en el dominio, no en la web.**
   - Decisión: `poker.ParseValue(raw string) (int, error)`: vacío → `ErrValueRequired`; no canónico ("05",
     " 5", "5.0", "abc") o fuera de la serie → `ErrValueNotInSeries`.
   - Motivo: una sola fuente de la regla; la web solo traduce el error.

8. **La serie se define en `src/poker` y se prueba contra HU-02.**
   - Decisión: `poker.Series = []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}`. `TestSeries_CoincideConStoryPoints`
     comprueba con `backlog.ValidateStoryPoints` que cada valor es válido y que 0, 4 y 90 no lo son.
   - Alternativa descartada: exportar el mapa de `src/backlog` (exige editar una historia ajena).

9. **Identificadores inválidos sin consultar la base.**
   - Decisión: `historia` y `ronda` se leen con `parseID`; 0 (ausente, no numérico, fuera de rango) se trata
     como inexistente (404) y el servicio no llama al almacén.
   - Motivo: igual que HU-19; no confunde un valor manipulado con una caída de la base.

10. **Otro proyecto = inexistente (404).**
    - Motivo: todas las consultas filtran por el proyecto del contexto; no se revela si el identificador existe
      en otro proyecto.

11. **Códigos de respuesta.** 303 éxito con `SetFlash`; 422 valor vacío o fuera de la serie; 409 ronda cerrada,
    historia con ronda abierta o que ya tuvo una; 404 ronda o historia inexistente; 500 falla al guardar con
    `MsgOperationFailed` (el detalle va al registro del servidor). La lectura fallida usa `Renderer.Fail`.

12. **Selector con todas las historias del proyecto** (respuesta del 27/10). El selector muestra el título y el
    estado; los escenarios 9 y 10 se reproducen desde la pantalla.

13. **Todos los integrantes con "Votó" / "Sin votar"** (respuesta del 27/10). La lista sale de
    `project_members` actual, ordenada por nombre. Consecuencia: un integrante quitado del proyecto deja de verse
    aunque su voto siga guardado (HU-05b decide si lo muestra al revelar).

14. **Rondas cerradas fuera de la pantalla.** HU-05a no crea rondas cerradas; mostrarlas es de HU-05b.

15. **Referencias a integrantes sin cascada.** `poker_rounds.opened_by` y `poker_votes.member_id` referencian
    `members(id)` sin `ON DELETE CASCADE`, como `effort_logs`. Las rondas caen en cascada con la historia y con
    el proyecto; los votos, con la ronda. Los fixtures borran historias → proyecto → integrantes.

16. **Sin token en el formulario.** HU-20b protege con `CrossOriginGuard` (middleware global).

17. **Mensajes en un único archivo** `src/web/poker_messages.go`, con prefijo `MsgPoker`. Los errores del
    dominio tienen textos técnicos que no se muestran; la web los traduce.

18. **Integrante en el servicio.** `Service` rechaza `memberID == 0` con `ErrMemberRequired` como defensa; desde
    la web no ocurre porque `RequireContext` redirige antes (escenario 12).

## Preguntas abiertas

Ninguna. Las tres dudas de la spec se respondieron el 27/10 (ver "Clarifications" en `spec.md`).
