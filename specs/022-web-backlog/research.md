# Research: Interfaz web del Product Backlog

## Decision 1: Una sola pantalla con formulario y lista mínima

- **Decision**: `/backlog` muestra el formulario de alta y la lista de historias del proyecto activo.
  El alta hace `POST /backlog` y redirige con 303 a la misma dirección.
- **Rationale**: El criterio "la historia aparece en el Backlog" necesita una lista para poder
  verificarse, y el listado del Backlog (HU-13) no está en el Sprint 2. Una sola pantalla evita una
  navegación extra y deja resuelto el estado vacío con el formulario a la vista, igual que HU-21.
- **Alternatives considered**: Depender de HU-13 para ver la lista (descartada: no está en el sprint);
  pantallas separadas de lista y de alta (descartada: agrega una ruta sin aportar a los criterios).

## Decision 2: Catálogo de lectura propio de HU-22

- **Decision**: Interfaz `StoryCatalog` con `ListStories(ctx, projectID)` y su adaptador
  `PostgreSQLStoryCatalog`, dentro de HU-22. Devuelve título, prioridad, Story Points, horas
  estimadas, estado y `id`, ordenado por `id` descendente.
- **Rationale**: El store de HU-02 está pensado para el alta y la lectura de una historia, no para
  listar un proyecto. Un catálogo de solo lectura evita ampliar el núcleo de dominio y deja a HU-13
  agregar orden por prioridad, filtro y edición.
- **Alternatives considered**: Ampliar el store de HU-02 (descartada: mezcla lectura de pantalla con
  el dominio); esperar HU-13 (descartada: no está en el sprint).

## Decision 3: Criterios dinámicos sin JavaScript

- **Decision**: El formulario tiene un campo por criterio (`criterio`, repetido). "Agregar criterio"
  y "Quitar" son botones de envío (`accion=agregar_criterio` y `accion=quitar_criterio:N`) que
  vuelven a mostrar el formulario con los datos escritos y sin guardar.
- **Rationale**: HU-20a no define JavaScript; con envíos del formulario el comportamiento se prueba
  con BDD automatizado, funciona sin scripts y sigue la política de recursos locales.
- **Alternatives considered**: JavaScript para agregar campos (descartada: solo se verifica a mano y
  agrega un recurso nuevo); un único campo de texto con un criterio por línea (descartada: se aparta
  de lo que pide el criterio de aceptación).

## Decision 4: Enter guarda, no agrega ni quita

- **Decision**: El primer botón de envío del formulario en el orden del documento es "Guardar"
  (visualmente oculto con una clase que no use `display: none`), y los botones de agregar y quitar
  van después.
- **Rationale**: Al presionar Enter en un campo, el navegador activa el primer botón de envío; si
  fuera "Quitar" se borrarían criterios sin que la persona lo pida.
- **Alternatives considered**: Dejar los botones en el orden visual (descartada: Enter quitaría el
  primer criterio); `display: none` en el botón oculto (descartada: algunos navegadores lo ignoran
  para el envío implícito).

## Decision 5: Criterios en blanco se ignoran

- **Decision**: Al guardar se descartan los criterios vacíos o con solo espacios; si no queda
  ninguno, el error de criterios de HU-02 se muestra junto al grupo de criterios.
- **Rationale**: Evita errores por campos que quedaron de más al usar "Agregar criterio" y deja a
  HU-02 la regla de "al menos un criterio".
- **Alternatives considered**: Rechazar si hay algún campo vacío (descartada: obliga a quitar a mano
  los campos sobrantes).

## Decision 6: El proyecto sale del contexto del servidor

- **Decision**: La sección se registra con `RequiresProject: true`; el handler toma el proyecto de
  `ActiveContextFrom(r.Context())` y descarta cualquier identificador de proyecto del formulario.
- **Rationale**: Evita crear historias en un proyecto distinto del activo con un envío manipulado y
  reutiliza la redirección de HU-20b cuando no hay proyecto.
- **Sobre "cookie expirada"**: las cookies de contexto de HU-20b son de sesión y no expiran por
  tiempo; lo que ocurre es que no haya proyecto activo válido (sesión nueva, proyecto eliminado o
  cookie inválida). HU-20b ya lo resuelve; HU-22 solo verifica que el guardado quede bloqueado.
- **Limitación aceptada**: en ese caso los datos del formulario no se conservan.

## Decision 7: Sin Story Points es una advertencia, no un error

- **Decision**: Sin Story Points la historia se crea y, como después se redirige, la advertencia se
  muestra con `SetFlash` de tipo advertencia. Como el mensaje de una sola vista es único, el texto
  combina la confirmación y la advertencia: "Historia creada correctamente. Falta completar la
  estimación de Story Points" (propuesta del equipo; si HU-02 define otro texto, se usa ese). Sin
  horas estimadas no hay advertencia: la columna dice "Sin estimar".
- **Rationale**: HU-02 permite crearla igual; la advertencia no puede quedar "junto al campo" porque
  el formulario ya no se muestra. En la lista, la historia aparece con "Sin estimar".
- **A verificar** (T001): cómo devuelve HU-02 la advertencia y su texto.

## Decision 8: No duplicar las reglas de HU-02

- **Decision**: El handler solo arma el envío (criterios, acciones, Story Points vacíos) y traduce los
  errores de HU-02 a campos (ver tabla en `plan.md`). La obligatoriedad de título, descripción,
  prioridad y criterios, los valores válidos y el estado inicial los decide el servicio.
- **Rationale**: Mantiene una única fuente de reglas y mensajes.
- **Alternatives considered**: Revalidar en el handler (descartada: dos copias que pueden divergir).

## Decision 9: Opciones del formulario tomadas de HU-02

- **Decision**: Las opciones de prioridad y de Story Points se obtienen de los valores válidos que
  expone HU-02 y no se escriben a mano en la plantilla. Si HU-02 no los expone como lista, se agrega
  una constante en el paquete web que coincida con sus reglas y una prueba que las compare.
- **Rationale**: Evita que la plantilla acepte valores que HU-02 rechaza, o al revés.
- **Confirmado en la migración `002`**: prioridad es `Alta`, `Media` o `Baja`, y Story Points es una
  lista cerrada (1, 2, 3, 5, 8, 13, 21, 34, 55, 89), así que ambos van como selectores. Una prueba
  verifica que HU-02 acepta cada opción y rechaza un valor fuera de la lista (T028).

## Decision 10: Códigos de respuesta

- **Decision**: Envío válido: 303 a `/backlog`. Agregar o quitar un criterio: 200 con el formulario.
  Error de validación: 422 con el formulario y los datos. Falla de guardado: 500 con el formulario,
  los datos y `MsgOperationFailed`. Falla de lectura de la lista: `Fail` de HU-20a. Sin proyecto
  activo: 303 de HU-20b a la selección.
- **Rationale**: Distingue los casos para las pruebas y el navegador sin cambiar lo que ve la persona.

## Decision 11: Lista simple y "Sin estimar"

- **Decision**: La lista muestra identificador, título, prioridad, Story Points, horas estimadas y
  estado, ordenada por `id` descendente, sin paginación. Un valor ausente de Story Points o de horas
  se muestra "Sin estimar".
- **Rationale**: Es lo mínimo para verificar el alta; cero sería engañoso para una historia sin
  estimar. El enunciado pide que cada historia muestre su identificador y HU-23 lo necesita para
  elegir historias. El orden por `id` descendente es provisional y no es un criterio de aceptación:
  HU-13 define el orden por prioridad y los filtros y reemplaza la prueba de orden.
- **Alternatives considered**: Ordenar por prioridad ahora (descartada: es alcance de HU-13).

## Decision 12: Estrategia de pruebas

- **Decision**: Pruebas unitarias del armado del formulario; pruebas de handler con `httptest` y
  `cookiejar` contra el servicio de HU-02 y PostgreSQL reales, eligiendo el proyecto activo por el
  flujo de HU-20b; un almacenamiento que falla solo para simular la falla de guardado y de lectura;
  integración de `PostgreSQLStoryCatalog` con fixtures propias; y escenarios BDD.
- **Rationale**: Verifica el comportamiento real, incluido el aislamiento entre proyectos y la
  redirección sin contexto.
- **Alternatives considered**: Inyectar el contexto en la solicitud a mano (descartada: no prueba la
  integración con HU-20b).

## Decision 13: Horas estimadas opcionales en el alta

- **Decision**: El formulario incluye un campo opcional "Horas estimadas" (`horas_estimadas`). Se pasa
  a HU-02 como dato opcional; vacío significa sin estimar.
- **Rationale**: La columna `estimated_hours` ya es opcional en `user_stories` (migración `002`) y
  HU-07 la lee. Sin este campo ninguna pantalla del Sprint 2 permite cargarla (HU-17 no tiene
  interfaz) y la desviación de esfuerzo de HU-25 mostraría siempre "No disponible" en la demo.
- **Atención**: la columna es `NUMERIC(6,2)`, que redondea sin error un valor con más de dos
  decimales (1,555 pasa a 1,56). La validación tiene que hacerse antes de guardar, en el dominio y no en
  la base, que solo exige `estimated_hours > 0`. Si HU-02 no la hace, se agrega en el dominio, junto
  a su regla, para que haya una sola que HU-17 reutilice.
- **Regla de horas**: las horas ya forman parte del alta de HU-02, así que HU-22 no depende de
  HU-17; HU-17 reutiliza la misma regla y mensaje al modificarlas. El formulario acepta coma o punto como separador decimal, no admite separadores
  de miles y muestra el valor con coma.
- **Alternatives considered**: Una pantalla de interfaz separada para HU-17 (descartada: otra historia
  que no está en el sprint); no cargarlas desde la web (descartada por lo anterior).
- **A verificar** (T001): si HU-02 acepta y valida las horas estimadas al crear, y cómo trata la coma
  decimal.
- **Sin límite de largo en título y descripción**: las columnas son `TEXT`, por lo que no hay
  validación de largo. Un límite sería una decisión de producto aparte.

## Decision 14: Máximo de horas estimadas

- **Decision**: Se rechazan las horas mayores que 9999,99 con "Las horas estimadas no pueden superar
  9999,99" junto al campo.
- **Rationale**: La columna `estimated_hours` es `NUMERIC(6,2)`; una cifra mayor fallaría al guardar
  con un error genérico ("La operación no pudo completarse") que engaña a quien la ingresó. El tope
  se escribe en el dominio, junto a la regla de horas de HU-02, no en el handler, para que haya una
  sola regla.
- **Pendiente del equipo**: el valor es una propuesta (el máximo de la columna); el equipo puede
  fijar un tope de negocio menor.
- **Alternatives considered**: Dejar que la base falle (descartada: mensaje engañoso); validar solo
  en el handler (descartada: dos copias de la regla).

## Decision 15: Quitar criterios

- **Decision**: "Quitar" elimina el campo de criterio elegido; con un solo campo restante no hace
  nada visible y el formulario se vuelve a mostrar con los datos escritos.
- **Rationale**: Una historia sin campo de criterio no se podría completar sin recargar la página, y
  HU-02 exige al menos un criterio.
- **Alternatives considered**: Permitir quitar todos los campos (descartada: deja un formulario sin
  forma de agregar el primer criterio sin una acción extra).
