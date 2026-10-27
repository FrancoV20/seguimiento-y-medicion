# Research: Actualización de estado de historias (Tablero)

## Decision 1: Pantalla propia "Tablero"

- **Decision**: Una sección `/tablero` separada de "Sprints" (HU-23).
- **Rationale**: Las acciones por historia (Comenzar, Completar, Reabrir) son de otro rol que planificar
  el Sprint, y una pantalla propia evita editar `sprints.html` y sus handlers mientras HU-23 se
  implementa en paralelo.
- **Alternatives considered**: Agregar los botones a `/sprints` (descartada: conflicto entre ramas).

## Decision 2: Regla y servicio en el dominio, sin tocar HU-04

- **Decision**: Archivos nuevos en `src/sprint` con la función pura `CanTransition(origen, destino)`, el
  servicio `ChangeStatus` y el puerto `StoryStatusStore`.
- **Rationale**: La lógica de negocio va fuera de la capa web (DoD, constitución II) y un puerto propio
  evita editar el store de HU-04, que HU-23 también toca (T047 de HU-23).
- **Alternatives considered**: Regla en el handler (descartada: dos copias posibles); ampliar el store de
  HU-04 (descartada: choque con HU-23).

## Decision 3: Story Points completados calculados

- **Decision**: Se calculan al leer: planificados = suma de todas las historias del Sprint; completados
  = suma de las historias "Hecho". Una historia sin Story Points cuenta 0.
- **Rationale**: No hay un contador que pueda duplicarse o quedar desactualizado al reabrir o enviar dos
  veces; coincide con la definición de HU-10 y de HU-07.

## Decision 4: Completar sin horas, permitido con advertencia

- **Decision**: Se permite y el mensaje de éxito incluye "Advertencia: no tiene horas reales
  registradas".
- **Rationale**: Resuelve el "(a definir)" del texto original, que el DoD no admite antes de
  implementar. No frena el avance del equipo y HU-07 ya maneja datos incompletos (cálculo parcial).
- **Riesgo aceptado**: una historia en "Hecho" sin horas distorsiona la desviación; la advertencia lo hace
  visible.
- **Alternatives considered**: Bloquear (descartada: obliga a cargar horas antes de poder avanzar, y
  HU-24 todavía no existe).
- **Estado**: decisión del equipo, confirmada.

## Decision 5: Cambio atómico con estado esperado

- **Decision**: Una sola sentencia `UPDATE … WHERE id = $1 AND status = $2 AND EXISTS (asignación a un
  Sprint activo)`; si afecta 0 filas, el servicio responde `ErrStatusChanged`.
- **Rationale**: Dos personas pueden presionar a la vez; la comparación atómica evita pisar un cambio
  ajeno y no deja estados intermedios.

## Decision 6: Orden de validación

- **Decision**: (1) la historia existe en el proyecto, (2) está en un Sprint activo, (3) ya está en el
  destino, (4) si llegó `estado_actual`, coincide con el real, (5) la transición real → destino está
  permitida, (6) el estado real es el origen de la acción, (7) cambio atómico.
- **Rationale**: Cada rechazo tiene un motivo único y un caso fuera del Sprint nunca se confunde con uno
  de transición; marcar dos veces es idempotente y no un error.
  El paso 6 cierra el caso de `comenzar` y `reabrir`, que comparten destino "En progreso": cada acción
  exige su origen y, si no coincide, se rechaza con el mensaje de estado desactualizado, sin mensajes
  nuevos.

## Decision 7: Acciones como identificadores

- **Decision**: El formulario envía `accion` (`comenzar`, `completar`, `reabrir`) y el estado que se veía
  (`estado_actual`), no el estado destino.
- **Rationale**: El navegador no elige el destino; el servidor lo deduce de la acción y rechaza todo lo
  demás. `estado_actual` permite detectar un tablero desactualizado.
  Cada acción lleva implícitos su origen y su destino; el botón que se presiona implica el estado que
  la persona veía.

## Decision 8: Sin identificación de integrante

- **Decision**: Cambiar el estado no exige integrante activo.
- **Rationale**: No hay roles ni autenticación (decisión del equipo de resolverlo después del Sprint 2) y
  HU-20b solo exige integrante para cargar esfuerzo y votar.

## Decision 9: Catálogo de lectura propio

- **Decision**: `BoardCatalog` con `ActiveBoard(ctx, projectID)` y su adaptador `PostgreSQLBoardCatalog`.
- **Rationale**: Solo lectura, sin tocar HU-04 ni HU-23. Calcula las acciones con `NextAction` del
  dominio y el progreso con una función pura.
- **Alternatives considered**: Reutilizar `SprintCatalog` de HU-23 (descartada: otra rama, otro
  contrato).

## Decision 10: Códigos de respuesta

- **Decision**: Cambio válido o estado ya aplicado: 303. Rechazo por Sprint, transición o estado
  desactualizado: 409. Historia o acción inválida: 422. Falla de guardado: 500. Falla de lectura:
  `Fail` de HU-20a. Sin proyecto: 303 de HU-20b.
- **Rationale**: Distingue los casos para las pruebas y el navegador.

## Decision 11: Estrategia de pruebas

- **Decision**: Pruebas unitarias de la regla; pruebas del servicio, de los adaptadores y de handler
  contra PostgreSQL real, con fixtures propias (dos proyectos; historias en cada estado; Sprint activo,
  Sprint finalizado y sin Sprint; horas); un almacenamiento que falla solo para simular la falla de
  guardado y de lectura; BDD.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada: se borran primero las
  historias y después el proyecto. Las pruebas no dependen de HU-22 ni de HU-23.