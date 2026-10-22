# Research: Cierre de Sprint

## Decision 1: Pantalla propia con resumen previo

- **Decision**: Una sección `/cierre` con un resumen (completadas, las que volverán al Backlog, Story
  Points) y el botón "Cerrar Sprint".
- **Rationale**: El cierre no se puede deshacer y devuelve historias al Backlog; ver el efecto antes de
  presionar evita cierres por error. Una pantalla propia no toca los templates de HU-14 ni de HU-23.
- **Alternatives considered**: Botón en `/sprints` o en `/tablero` (descartada: edita pantallas ajenas y
  no muestra el resumen).

## Decision 2: Las no completadas vuelven a "Pendiente" y pierden la asignación

- **Decision**: Las historias en "En Sprint" o "En progreso" pasan a "Pendiente" y se elimina su fila de
  `sprint_story_assignments`; las "Hecho" conservan estado y asignación.
- **Rationale**: Voto del equipo (21/10/2026): "vuelven al Product Backlog para poder asignarse a otro
  Sprint". La asignación es única por historia, así que hay que quitarla para poder asignarla de nuevo,
  y "Pendiente" es lo que HU-04 considera elegible.

## Decision 3: Registro del Sprint al cerrar

- **Decision**: En el mismo cierre se insertan en `sprint_story_results(sprint_id, story_id, final_status,
  story_points)` todas las historias del Sprint, antes de quitar asignaciones.
- **Rationale**: Al desasignar, el Sprint cerrado perdería qué había planificado. Sin esto, HU-10
  (porcentaje de historias completadas y SP planificados), la Sprint Review (SP planificados vs
  completados) y HU-16 (historias no completadas) quedarían mal, y no se puede reconstruir después. Es
  "congelar los datos" de la descripción de la historia.
- **Alternatives considered**: No guardar nada (descartada: pérdida irrecuperable); conservar las
  asignaciones y relajar la unicidad (descartada: cambia HU-04 y su restricción); guardar el último Sprint
  en la historia (descartada: una historia puede pasar por varios Sprints).

## Decision 4: Cierre transaccional todo o nada

- **Decision**: Una transacción: bloquear el Sprint (`FOR UPDATE`) y sus historias, comprobar que sigue
  activo, insertar el registro, devolver las no completadas y pasar el Sprint a "Finalizado".
- **Rationale**: FR-012 exige que no queden cambios parciales; bloquear las historias evita que un cambio
  de estado de HU-14 llegue a mitad del cierre (su sentencia exige un Sprint activo y falla al terminar
  el cierre). Bloquear el Sprint hace que dos cierres simultáneos se serialicen.

## Decision 5: Validación de HU-07 antes de la transacción

- **Decision**: El servicio invoca la validación de HU-07 antes de cerrar; si devuelve historias a
  corregir, responde `ErrCloseBlocked` sin tocar nada.
- **Rationale**: HU-07 ya implementa la regla ("HU-15 solo la invoca"). Se evita duplicarla. Los datos
  inválidos no se corrigen durante el cierre, así que no hace falta leer dentro de la transacción;
  se acepta la carrera de que alguien cargue horas justo en medio.
- **Alternatives considered**: Revalidar dentro de la transacción (descartada: duplicaría la lectura de
  HU-07).

## Decision 6: El Sprint viene del formulario y se valida

- **Decision**: `POST /cierre` envía `sprint_id` (el que se veía); el servicio comprueba que pertenece al
  proyecto activo y que está activo.
- **Rationale**: Si el Sprint saliera solo del servidor, una pantalla vieja podría cerrar sin querer un
  Sprint nuevo creado en el medio. Con el identificador, ese envío se rechaza como "El Sprint ya está
  cerrado".

## Decision 7: Reutilizar el tablero de HU-14 para el resumen

- **Decision**: El resumen lee `BoardCatalog.ActiveBoard` (historias por estado y Story Points).
- **Rationale**: Ya calcula los grupos y los puntos; HU-15 depende de HU-14, así que está en `main`.
- **Alternatives considered**: Catálogo propio (descartada: duplicación).

## Decision 8: Velocidad no se muestra ni se calcula acá

- **Decision**: El mensaje de éxito no incluye velocidad; el cálculo es de HU-07 y la pantalla de
  métricas, de HU-25. Las pruebas verifican con HU-07 que la velocidad es cero cuando no hay "Hecho".
- **Rationale**: Mantiene a HU-15 desacoplada del cálculo y de su pantalla.

## Decision 9: Regla de carga de esfuerzo posterior

- **Decision**: Se verifica que HU-03 rechaza horas sobre historias fuera del Sprint activo (incluidas las
  del Sprint finalizado y las devueltas); si no lo hace, se agrega ahí con su prueba.
- **Rationale**: El escenario original lo atribuye a HU-03 ("regla de HU-03"); sin ella, el Sprint
  cerrado podría recibir horas y cambiar su desviación.

## Decision 10: Sin condicionar al calendario

- **Decision**: Se puede cerrar antes o después de la fecha de fin.
- **Rationale**: Ningún criterio lo prohíbe y el Scrum Master decide cuándo termina el Sprint.

## Decision 11: Códigos de respuesta

- **Decision**: Cierre válido: 303. Bloqueo de HU-07 o Sprint ya cerrado: 409. Sprint inválido: 422. Falla
  de guardado: 500. Falla de lectura: `Fail` de HU-20a. Sin proyecto: 303 de HU-20b.
- **Rationale**: Distingue los casos para las pruebas y el navegador.

## Decision 12: Estrategia de pruebas

- **Decision**: Pruebas del servicio, del adaptador y de handler contra PostgreSQL real con fixtures
  propias (dos proyectos; Sprint con historias "Hecho", "En progreso" y "En Sprint"; Sprint ya
  finalizado); la validación de HU-07 con datos inválidos se prueba como HU-07 prueba ese caso (se
  confirma en T001); un almacenamiento que falla solo para simular la falla de guardado y de lectura;
  BDD.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada: se borran primero las
  historias y después el proyecto. La migración nueva usa `ON DELETE CASCADE` hacia `sprints` y
  `user_stories`.