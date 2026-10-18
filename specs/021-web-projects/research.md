# Research: Interfaz web de proyectos

## Decision 1: Una sola pantalla con formulario y lista

- **Decision**: `/proyectos` muestra el formulario de alta y la lista de proyectos. El alta hace
  `POST /proyectos` y redirige con 303 a la misma dirección.
- **Rationale**: La historia describe un usuario "en Gestión de Proyectos" que completa el
  formulario y ve el proyecto en la lista; una sola pantalla evita una navegación extra y deja el
  estado vacío resuelto con el formulario a la vista.
- **Alternatives considered**: Pantallas separadas de lista y de alta (descartada: agrega una ruta
  y un clic sin aportar a los criterios de aceptación).

## Decision 2: Catálogo de integrantes y lista propios de HU-21

- **Decision**: Interfaz `ProjectCatalog` con `ListProjects` y `ListMembers`, y su adaptador
  `PostgreSQLProjectCatalog`, dentro de HU-21.
- **Rationale**: `members` es un catálogo global que ninguna historia lee completo. El
  `ContextStore` de HU-20b lista los integrantes de un proyecto, no todos, y el equipo decidió no
  ampliar esa historia. El store de HU-01 solo comprueba que un integrante existe.
- **Alternatives considered**: Agregar campos y métodos a `ContextStore` (descartada: modifica
  HU-20b); ampliar el store de HU-01 (descartada: mezcla lectura de pantalla con el núcleo de
  dominio).

## Decision 3: Los integrantes se cargan con un script semilla

- **Decision**: `db/seeds/members.sql` inserta los cinco integrantes del equipo y no duplica si se
  corre dos veces (`INSERT ... WHERE NOT EXISTS`). No es una migración y no consume número.
- **Rationale**: El esquema de HU-01 aclara que el alta de integrantes se gestiona fuera de esa
  historia, y ninguna historia del Sprint 2 los crea. La semilla desbloquea el formulario sin
  agregar alcance.
- **Alternatives considered**: Alta de integrantes dentro del formulario (descartada: exige un
  método de alta en HU-01, una regla contra nombres repetidos, probablemente una migración, y una
  transacción con el proyecto); historia nueva de alta de integrantes (queda en el backlog para un
  sprint posterior, porque el TP pide registrar integrantes).
- **Nota**: `display_name` no es único, por eso la semilla compara por nombre antes de insertar.

## Decision 4: El handler no duplica las reglas de HU-01

- **Decision**: El handler solo valida lo que es propio del envío web: el formato de las fechas,
  que los identificadores de integrantes sean numéricos y los repetidos. El resto (nombre
  obligatorio, fechas coherentes, integrantes existentes) lo decide `CreateProject`.
- **Rationale**: Mantiene una única fuente de las reglas y los mensajes exactos que pide el
  criterio de aceptación.
- **Alternatives considered**: Revalidar todo en el handler (descartada: dos copias de las reglas
  que pueden divergir).

## Decision 5: Mapeo de errores a campos

- **Decision**: Los errores de validación de HU-01 se traducen a un mensaje junto al campo
  correspondiente (ver tabla en `plan.md`); cualquier otro error se muestra como
  `MsgOperationFailed`.
- **Rationale**: Cumple "cada mensaje junto a su campo" y evita exponer detalles internos.
- **A verificar** (T001): los tipos de error y los mensajes reales de HU-01; si HU-01 devuelve un
  solo error por vez o todos juntos, para mostrarlos en una misma respuesta.

## Decision 6: Códigos de respuesta

- **Decision**: Envío válido: 303 a `/proyectos`. Error de validación: 422 con el formulario y los
  datos. Falla de guardado: 500 con el formulario, los datos y `MsgOperationFailed`. Falla de
  lectura de la lista o los integrantes: `Fail` de HU-20a (503 o 500).
- **Rationale**: Distingue los casos para las pruebas y para el navegador sin cambiar lo que ve la
  persona.

## Decision 7: Fechas con `AAAA-MM-DD` y sin zona horaria

- **Decision**: Parsear con `time.Parse("2006-01-02", valor)` y tratar el resultado como fecha de
  calendario; mostrar las fechas como `dd/mm/aaaa`.
- **Rationale**: El control de fecha del navegador envía `AAAA-MM-DD`; evita corrimientos por zona
  horaria y coincide con las columnas `DATE`.
- **Alternatives considered**: Aceptar varios formatos (descartada: ambigüedad entre día y mes).

## Decision 8: Identificadores de integrantes sin repetidos

- **Decision**: Quitar identificadores repetidos antes de invocar el alta.
- **Rationale**: La relación proyecto-integrante tiene clave primaria compuesta; un repetido haría
  fallar el guardado por un dato que la persona no puede ver.
- **Alternatives considered**: Rechazar con error (descartada: no aporta valor al usuario).

## Decision 9: Lista simple, más recientes primero

- **Decision**: `ListProjects` devuelve nombre, fechas y cantidad de integrantes, ordenados por
  `id` descendente, sin estado y sin paginación.
- **Rationale**: Es lo que pide el criterio de aceptación. `projects.status` vale `'Activo'` por
  defecto y no corresponde al estado de HU-12, que se calcula con las fechas.
- **Alternatives considered**: Ordenar por fecha de inicio (descartada: el alta reciente no
  aparecería arriba si su fecha es anterior); paginar (descartada: no hay requisito).

## Decision 10: Sin botón de selección en la lista

- **Decision**: La lista no ofrece "Seleccionar" ni reutiliza `project_picker.html`.
- **Rationale**: La selección del proyecto activo vive en la pantalla de HU-20b; duplicarla
  acoplaría HU-21 a US3 de esa historia, que no es una dependencia acordada.
- **Alternatives considered**: Botón de selección en cada fila (descartada: amplía alcance y
  dependencias).

## Decision 11: Estrategia de pruebas

- **Decision**: Pruebas unitarias del parseo del formulario; pruebas de handler con `httptest`
  contra el `ProjectService` y PostgreSQL reales; un `ProjectStore` que falla solo para simular la
  falla de guardado y la de lectura; integración de `PostgreSQLProjectCatalog` con fixtures propias
  (no dependen de la semilla); y escenarios BDD.
- **Rationale**: Verifica el comportamiento real y mantiene el ciclo TDD rápido, usando un
  almacenamiento que falla únicamente para lo que no se puede provocar de otra forma.
- **Alternatives considered**: Store en memoria para todo (descartada: no valida las consultas ni
  el principio VI).

## Decision 12: Criterios de usabilidad de HU-01

- **Decision**: SC-003 y SC-004 se miden a mano en la Sprint Review con una prueba corta (ver
  `quickstart.md`) y se guardan en `evidencia-usabilidad.txt`.
- **Rationale**: Requieren personas, no pruebas automáticas; HU-01 los difirió hasta que existiera
  esta pantalla.
- **Pendiente**: confirmar los umbrales (90 % y 2 minutos son una propuesta) contra la spec de
  HU-01.

## Decision 13: Nombre de más de 255 caracteres (extra)

- **Decision**: Si HU-01 no limita el largo del nombre, rechazarlo en el handler con un mensaje
  junto al campo; es un extra que no es criterio de aceptación.
- **Rationale**: La columna es `VARCHAR(255)`; sin esta validación el error de la base se mostraría
  como "La operación no pudo completarse", que confunde.
- **Alternatives considered**: Dejar que falle el guardado (aceptable si el equipo prioriza el
  alcance, porque ya cae en el mensaje general sin exponer detalles).
