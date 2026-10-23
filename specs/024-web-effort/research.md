# Research: Interfaz web de registro de esfuerzo

## Decision 1: Una sola pantalla con lista, formulario e historial

- **Decision**: Una sección `/esfuerzo` que muestra las historias del Sprint activo con sus horas acumuladas, el
  formulario y el historial de la historia elegida con `?historia=ID`.
- **Rationale**: Sin JavaScript, un enlace por historia es la forma más simple de elegir cuál ver; al guardar, la
  redirección vuelve a esa misma historia y se ve el acumulado actualizado.
- **Alternatives considered**: Una pantalla de historial aparte (descartada: más rutas para lo mismo); cargar el
  historial con JavaScript (descartada: la pantalla funciona sin JavaScript).

## Decision 2: El integrante sale de la identificación, y la sección lo exige

- **Decision**: El formulario no tiene campo de integrante; el registro usa el integrante identificado de HU-20b, y
  la sección se declara con proyecto e integrante requeridos.
- **Rationale**: HU-20b define que las acciones "(esfuerzo, votos) quedan asociadas a mi nombre" y pide
  identificarse antes de cargar esfuerzo. Con la sección completa protegida, el formulario nunca se muestra sin
  poder enviarse.
- **Alternatives considered**: Exigir integrante solo al enviar (descartada: muestra un formulario que no se puede
  usar); un campo de integrante en el formulario (descartada: permitiría cargar horas a nombre de otro).

## Decision 3: Las reglas de HU-03 no se copian

- **Decision**: La capa web solo lee los datos (fecha, horas con coma o punto) y pide a HU-03 las violaciones;
  muestra sus mensajes junto a cada campo.
- **Rationale**: "Los mensajes exactos de HU-03" implica una única fuente. Copiar las reglas las desincroniza.
- **Alternatives considered**: Validar en el handler (descartada: duplica reglas).

## Decision 4: Todos los errores juntos requieren una validación completa en HU-03

- **Decision**: Se usa una función de HU-03 que devuelve todas las violaciones. Si HU-03 solo devuelve la primera, T010
  agrega esa función sin cambiar el comportamiento del registro.
- **Rationale**: El escenario pide ver todos los mensajes a la vez; con solo el primero habría que corregir de a uno.
- **Alternatives considered**: Llamar varias veces al servicio con datos parciales (descartada: frágil y puede
  registrar por error); dejar un solo error (descartada: incumple el criterio).

## Decision 5: Selector con todas las historias del Sprint activo

- **Decision**: Se ofrecen las historias asignadas al Sprint activo del proyecto, sea cual sea su estado ("En
  Sprint", "En progreso" o "Hecho"), con el estado en la etiqueta. T001 confirma que HU-03 acepta las mismas.
- **Rationale**: El criterio dice "solo las del Sprint activo" y HU-03, "historia asignada en el Sprint activo". Una
  historia "Hecho" puede recibir horas pendientes antes del cierre. El selector debe ofrecer exactamente lo que
  HU-03 acepta.
- **Alternatives considered**: Solo "En progreso" (descartada: no lo dice ninguna historia y cambia HU-03).

## Decision 6: Catálogo de solo lectura propio

- **Decision**: `EffortCatalog` con `ActiveSprintStories` y `StoryHistory`, adaptador `PostgreSQLEffortCatalog` en
  `src/web`.
- **Rationale**: Mantiene la lectura de pantalla separada de las reglas de HU-03 y evita editar archivos de otras
  historias, igual que el catálogo de HU-23.

## Decision 7: Historia y proyecto verificados antes de registrar

- **Decision**: El handler comprueba que `historia` esté entre las del Sprint activo del proyecto activo antes de
  llamar a HU-03, y HU-03 vuelve a validar su propia regla.
- **Rationale**: Evita cargar horas en una historia de otro proyecto aunque HU-03 solo verificara el Sprint de la
  historia. Si el Sprint se cierra en el medio, HU-03 lo rechaza y se muestra junto al campo de historia.

## Decision 8: Un `historia` inválido en la dirección se ignora

- **Decision**: Un `historia` no numérico, inexistente o fuera del Sprint activo en `GET` se trata como si no
  estuviera elegido y no muestra historial.
- **Rationale**: Es una consulta, no una acción; no necesita un mensaje y evita una pantalla de error por un enlace
  viejo.

## Decision 9: Fecha por defecto y zona horaria

- **Decision**: La fecha por defecto es la de hoy según el reloj del servidor, que se inyecta para poder probarla.
- **Rationale**: "Fecha (por defecto hoy)". No se valida contra el Sprint ni el futuro porque HU-03 no lo pide.

## Decision 10: Códigos de respuesta

- **Decision**: Carga válida: 303. Datos inválidos o historia no disponible: 422. Sin Sprint activo al enviar: 409.
  Falla de guardado: 500. Falla de lectura: `Fail` de HU-20a. Sin proyecto o integrante: 303 de HU-20b.
- **Rationale**: Distingue los casos para las pruebas y el navegador.

## Decision 11: Estrategia de pruebas

- **Decision**: Pruebas de handler y de integración contra PostgreSQL real con fixtures propias (dos proyectos, cada
  uno con un Sprint activo, uno finalizado, historias "En Sprint", "En progreso", "Hecho" y "Pendiente", y
  registros de dos integrantes); el servicio de HU-03 real; un almacenamiento que falla solo para simular la falla de
  guardado y de lectura; BDD.
- **Nota de fixtures**: `user_stories.project_id` no tiene borrado en cascada: se borran primero las historias y
  después el proyecto.