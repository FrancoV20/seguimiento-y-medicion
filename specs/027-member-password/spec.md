# Especificación: HU-27 – Identificación con Contraseña

**Rama**: `HU-27`
**Creada**: 2026-10-28
**Estado**: Propuesta (pendiente de aceptación del equipo)
**Constitución**: v1.1.0
**Prioridad**: Baja – Could Have (trabajo futuro si no llega; el equipo la integra al final) | **Valor**: 60 |
**Estimación**: 5 Story Points (sugerida)
**Depende de**: HU-20b y HU-26 (las dos en `main`). Se integra después de todas las demás historias, porque cambia
la identificación que usan sus pruebas.

## Historia de usuario

Como Integrante del proyecto, quiero identificarme con una contraseña, para que nadie pueda registrar esfuerzo,
votar ni hacer otras acciones a mi nombre.

## Clarifications

### Session 2026-10-28

Decisiones del equipo (dadas al pedir la especificación; no se volvieron a preguntar):

- Q: ¿Dónde viven las reglas de la contraseña? → A: En un paquete nuevo `src/auth`, con funciones puras
  `HashPassword(password) (string, error)`, `VerifyPassword(password, stored) (bool, error)` y
  `ValidateNewPassword(password, confirm) error`. Largo de 8 a 128 caracteres contados como runas. Mensajes: "La
  contraseña debe tener al menos 8 caracteres", "La contraseña no puede superar 128 caracteres" y "Las contraseñas no
  coinciden".
- Q: ¿Con qué algoritmo se guarda? → A: `crypto/pbkdf2` de la biblioteca estándar con SHA-256, sal aleatoria de 16
  bytes (`crypto/rand`), 600.000 iteraciones y clave de 32 bytes. Formato guardado
  `pbkdf2-sha256$600000$<sal base64>$<hash base64>`. Comparación con `crypto/subtle.ConstantTimeCompare`. Las
  iteraciones se leen del valor guardado. Sin dependencias nuevas: `go.mod` no cambia.
- Q: ¿Cómo funciona el bloqueo? → A: Un tipo en `src/auth` con reloj inyectable: 5 fallos seguidos por integrante lo
  bloquean 1 minuto desde el quinto fallo; un acierto reinicia el contador; seguro para uso concurrente
  (`sync.Mutex`). Vive en memoria y se reinicia con el servidor (limitación documentada).
- Q: ¿Qué cambia en la base? → A: Una migración nueva: `ALTER TABLE members ADD COLUMN password_hash TEXT NULL`
  (down: `DROP COLUMN`). El número es el siguiente libre en `main` al implementar (hoy la mayor es 011; HU-06 y HU-15
  toman las siguientes). T001 lo confirma y T038 lo vuelve a confirmar antes del PR.
- Q: ¿Cómo se lee y se guarda la contraseña? → A: Se amplía `ContextStore` de HU-20b con un método que lee el hash de
  un integrante del proyecto (dice si pertenece al proyecto y si tiene contraseña). Otro puerto con su adaptador
  guarda el hash; lo usa el comando. Los dos adaptadores son PostgreSQL y tienen pruebas de integración.
- Q: ¿Qué responde la pantalla "Identificarme" (`POST /contexto/integrante` con `integrante_id` y `contrasena`)? → A:
  - integrante que no es del proyecto: el comportamiento actual de HU-20b;
  - contraseña vacía: 422, "Ingresá tu contraseña" junto al campo;
  - integrante bloqueado: 429, "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar", sin verificar la
    contraseña;
  - sin contraseña configurada: 422, "<Nombre> todavía no tiene contraseña configurada. Pedile al Scrum Master que la
    configure";
  - contraseña incorrecta: 401, "Integrante o contraseña incorrectos"; cuenta como fallo;
  - contraseña correcta: la cookie firmada de HU-26 como hoy, `SetFlash` "Te identificaste como <Nombre>" y 303 al
    destino saneado con `SanitizeReturnTo`;
  - falla de lectura: `Renderer.Fail`, sin identificar.

  Ante cualquier rechazo se conserva el integrante elegido y el campo de contraseña vuelve vacío. El campo es
  `type="password"` con `autocomplete="current-password"` y nunca se vuelve a mostrar su valor.
- Q: ¿Cómo se sale? → A: `POST /contexto/salir` borra la cookie del integrante (no la del proyecto), hace `SetFlash`
  "Cerraste tu identificación" y responde 303 al inicio, o al destino saneado si viene uno. El botón "Salir" está en la
  estructura común, junto al nombre del integrante identificado. Lo protege el `CrossOriginGuard` global.
- Q: ¿Dónde puede aparecer la contraseña? → A: En ningún lado: ni en el registro del servidor, ni en mensajes, ni en
  la página, ni en errores. Una prueba lo verifica.
- Q: ¿Cómo se configura una contraseña? → A: Con el comando `src/cmd/set-password` (`-integrante N`, `DATABASE_URL`
  como el servidor). Pide la contraseña y la confirmación por la entrada estándar (se ven en pantalla: limitación
  documentada), valida con `ValidateNewPassword`, guarda el hash y muestra "Contraseña configurada para <Nombre>". Si
  el integrante no existe: "El integrante no existe". La lógica va en una función
  `run(args, stdin, stdout, getenv)` que se prueba sin base real, y además hay una prueba de integración.
- Q: ¿Qué pasa con las pruebas existentes? → A: Se actualizan las que se identifican por `/contexto/integrante` (T001
  las cuenta con `git grep`), con un ayudante que configura la contraseña de las fixtures y la envía. Se reemplazan
  las expectativas de HU-20b sobre "sin contraseña"; las demás no cambian.
- Q: ¿Qué queda fuera? → A: Roles (HU-28), cambiar o recuperar la contraseña desde la web, vencimiento de sesión e
  invalidar sesiones al cambiar la contraseña.

`/speckit-clarify` (28/10) no encontró dudas nuevas que cambien el alcance. Lo que el Issue y las decisiones dejaban
abierto se resolvió en "Casos límite" y "Supuestos y decisiones" (orden de las comprobaciones, qué cuenta como fallo,
qué pasa al vencer el bloqueo, la identificación previa ante un rechazo y la contraseña de Dani en las fixtures).

## Sección SDD

### Intención

Que solo quien conoce la contraseña de un integrante pueda actuar a su nombre. Hoy (HU-20b) identificarse es elegir
un nombre de la lista, y HU-26 solo impide falsificar la cookie: cualquiera puede elegir a Ana y votar por ella. HU-27
agrega la contraseña al paso de identificación, un bloqueo corto contra la prueba de contraseñas y un comando para que
el Scrum Master la configure.

### Alcance

- La pantalla "Identificarme" (`GET` y `POST /contexto/integrante`) pide integrante y contraseña.
- "Salir" (`POST /contexto/salir`) y su botón en la estructura común.
- Guardado de la contraseña con sal y derivación lenta; verificación en tiempo constante.
- Bloqueo en memoria: 5 fallos seguidos por integrante bloquean 1 minuto.
- Comando `go run ./src/cmd/set-password -integrante N` para configurar la contraseña.
- Migración que agrega `members.password_hash` (up y down).
- Actualización de las pruebas existentes que se identifican por `/contexto/integrante`.

### Fuera de alcance

- Roles y permisos (HU-28).
- Cambiar o recuperar la contraseña desde la web; pantalla para configurarla.
- Vencimiento de la sesión e invalidar sesiones abiertas al cambiar la contraseña.
- Ocultar lo que se escribe en el comando (necesitaría una dependencia nueva).
- Bloqueo persistente o compartido entre procesos; bloqueo por dirección IP.
- Normalización Unicode de la contraseña.

### Entradas

- Formulario "Identificarme": `integrante_id`, `contrasena` y `volver_a` (opcional).
- Formulario "Salir": `volver_a` (opcional).
- Comando: opción `-integrante N`, variable `DATABASE_URL` y dos líneas por la entrada estándar (contraseña y
  confirmación).

### Salidas

- Cookie firmada `ctx_integrante` (HU-26) al identificarse; borrada al salir.
- Mensajes de una sola vista ("Te identificaste como Ana", "Cerraste tu identificación") y páginas de rechazo con su
  código (401, 422, 429).
- Columna `members.password_hash` con el valor derivado, nunca la contraseña.
- Salida del comando ("Contraseña configurada para Caro" o el motivo del rechazo) y su código de salida.

### Reglas

1. Orden de las comprobaciones de `POST /contexto/integrante`: proyecto activo (HU-20b) → el integrante pertenece al
   proyecto → contraseña no vacía → integrante no bloqueado → tiene contraseña configurada → verificación.
2. Solo una contraseña incorrecta cuenta como fallo. No cuentan: integrante de otro proyecto, contraseña vacía,
   intento durante el bloqueo, integrante sin contraseña ni falla de lectura.
3. El quinto fallo seguido responde 401 y deja bloqueado al integrante; mientras dura el bloqueo (60 segundos desde
   el quinto fallo) todo intento responde 429 sin verificar. Al vencer, el contador vuelve a cero.
4. Un acierto reinicia el contador. Los fallos no vencen por tiempo: solo un acierto o el fin de un bloqueo los
   borran.
5. Un rechazo por contraseña (401, 422, 429) no cambia la identificación que ya tenía el navegador. El rechazo por
   integrante de otro proyecto la limpia, como hoy en HU-20b.
6. La contraseña no se recorta: los espacios forman parte de ella, en la web y en el comando. El comando solo quita
   el fin de línea (`\n` o `\r\n`).

### Restricciones

- Go 1.27.1, PostgreSQL y la biblioteca estándar; `go.mod` no cambia.
- Renderizado en el servidor, sin JavaScript.
- No cambian la firma de `RegisterContext` ni `src/cmd/server/main.go`.
- Se integra al final, después de todas las demás historias.

### Condiciones de error

| Situación | Respuesta | ¿Identificado? | ¿Cuenta como fallo? |
|---|---|---|---|
| Sin proyecto activo | 303 a elegir proyecto (HU-20b) | No | No |
| Integrante ausente, inválido o de otro proyecto | 303 al selector con el mensaje de HU-20b; se borra la cookie del integrante | No | No |
| Contraseña vacía | 422, "Ingresá tu contraseña" junto al campo | Sin cambios | No |
| Integrante bloqueado | 429, "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" | Sin cambios | No |
| Sin contraseña configurada | 422, "<Nombre> todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure" | Sin cambios | No |
| Contraseña incorrecta | 401, "Integrante o contraseña incorrectos" | Sin cambios | Sí |
| Base no disponible al leer | 503 de HU-20a, sin detalles | Sin cambios | No |
| Valor guardado mal formado | 500 de HU-20a, sin detalles | Sin cambios | No |
| `GET /contexto/salir` | 405 de HU-20a | — | — |
| Envío de otro origen | 403 de HU-20b | — | — |

## Datos de ejemplo

Del Issue, sin cambios:

- Proyecto «Alfa», integrantes Ana (contraseña "ana-clave-2026"), Beto (contraseña "beto-clave-2026") y Caro (sin
  contraseña configurada).
- Proyecto «Beta», integrante Dani.
- Bloqueo: 5 intentos fallidos seguidos bloquean al integrante durante 1 minuto.

Agregado para las pruebas (el Issue no lo da):

- Dani tiene la contraseña "dani-clave-2026", para que el escenario 6 envíe "su contraseña".
- En la prueba del escenario 9, y solo en ella, Beto se configura con "ana-clave-2026", la misma que Ana, porque el
  escenario pide dos integrantes con la misma contraseña.
- Contraseñas incorrectas: "otra-clave" (la del escenario 2).

## Escenarios BDD

Texto del Issue HU-27.

**Escenario 1: Identificación correcta**
Dado el proyecto «Alfa» elegido
Cuando elijo a Ana, ingreso "ana-clave-2026" y presiono "Identificarme"
Entonces veo "Te identificaste como Ana", su nombre queda visible en la estructura común y vuelvo a la pantalla que
había pedido

**Escenario 2: Contraseña incorrecta**
Dado el proyecto «Alfa» elegido
Cuando elijo a Ana e ingreso "otra-clave"
Entonces veo "Integrante o contraseña incorrectos", no quedo identificado, Ana sigue elegida y el campo de contraseña
aparece vacío

**Escenario 3: Contraseña vacía**
Dado el proyecto «Alfa» elegido
Cuando elijo a Beto y presiono "Identificarme" sin ingresar contraseña
Entonces veo "Ingresá tu contraseña" junto al campo y no quedo identificado

**Escenario 4: Integrante sin contraseña**
Dado que Caro todavía no tiene contraseña
Cuando la elijo e ingreso cualquier contraseña
Entonces veo "Caro todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure" y no quedo
identificado

**Escenario 5: Demasiados intentos fallidos**
Dado que se ingresó 5 veces seguidas una contraseña incorrecta para Ana
Cuando ingreso su contraseña correcta antes de que pase un minuto
Entonces veo "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" y no quedo identificado; pasado el
minuto, con la contraseña correcta me identifico

**Escenario 6: Integrante de otro proyecto**
Dado el proyecto «Alfa» elegido
Cuando se envía la identificación de Dani, de «Beta», con su contraseña
Entonces se rechaza con el mensaje de HU-20b y no quedo identificado

**Escenario 7: Salir**
Dado que estoy identificado como Ana
Cuando presiono "Salir"
Entonces veo "Cerraste tu identificación", mi nombre deja de verse y, si intento registrar esfuerzo o votar, se me
pide identificarme con contraseña

**Escenario 8: Cambio de proyecto**
Dado que estoy identificado como Ana en «Alfa»
Cuando cambio a otro proyecto y vuelvo a «Alfa»
Entonces la identificación se limpió y para volver a identificarme tengo que ingresar la contraseña

**Escenario 9: Contraseña protegida**
Dado que Ana y Beto tienen contraseñas configuradas
Cuando se revisa la base de datos, la página y el registro del servidor
Entonces ninguna contraseña aparece en texto plano y dos integrantes con la misma contraseña tienen valores guardados
distintos

**Escenario 10: Configurar una contraseña**
Dado que Caro no tiene contraseña
Cuando el Scrum Master ejecuta el comando de configuración para Caro e ingresa dos veces "caro-clave-2026"
Entonces ve "Contraseña configurada para Caro" y Caro puede identificarse con esa contraseña

**Escenario 11: Contraseña inválida al configurar**
Dado el comando de configuración para Caro
Cuando se ingresa una contraseña de menos de 8 caracteres o las dos contraseñas no coinciden
Entonces el comando informa el motivo ("La contraseña debe tener al menos 8 caracteres" o "Las contraseñas no
coinciden") y no cambia nada

**Escenario 12: Error al verificar**
Dado un fallo al leer los datos del integrante
Cuando intento identificarme
Entonces veo la página de error de la aplicación, sin detalles internos, y no quedo identificado

### Cuentas del bloqueo (escenario 5)

| Momento (desde el 1.er fallo) | Intento | Respuesta | Fallos seguidos | Bloqueado |
|---|---|---|---|---|
| 0 s a 4 s | "otra-clave" ×4 | 401 ×4 | 4 | No |
| 5 s | "otra-clave" | 401 | 5 | Sí, hasta 65 s |
| 30 s | "ana-clave-2026" | 429, no se verifica | 5 | Sí |
| 64 s | "ana-clave-2026" | 429 | 5 | Sí |
| 65 s | "ana-clave-2026" | 303, "Te identificaste como Ana" | 0 | No |

Con un acierto en el cuarto intento, el contador vuelve a 0 y hacen falta otros 5 fallos para bloquear.

## Casos límite

- **Rechazo con una identificación previa**: si el navegador ya estaba identificado como Beto y falla un intento como
  Ana (401, 422 o 429), sigue identificado como Beto (regla 5).
- **Contraseña con espacios o caracteres no ASCII**: los espacios cuentan y no se recortan; "ñ" o un emoji cuentan como
  un carácter. No hay normalización Unicode: la misma palabra escrita con otra composición de acentos es otra
  contraseña.
- **Contraseña de más de 128 caracteres al identificarse**: no se rechaza por largo; se verifica y, como no coincide,
  cuenta como fallo.
- **Valor guardado mal formado o con iteraciones fuera de rango** (por ejemplo, cargado a mano): página 500 sin
  detalles; no se identifica ni cuenta como fallo.
- **Varios intentos simultáneos**: el bloqueo es seguro para uso concurrente, pero la consulta y el registro del fallo
  son pasos separados; una ráfaga de envíos en paralelo puede verificar algunas contraseñas más antes de bloquear
  (limitación documentada).
- **Reinicio del servidor**: los contadores y los bloqueos se pierden (limitación documentada).
- **Bloqueo como molestia**: cualquiera puede bloquear a Ana 1 minuto con 5 intentos; el bloqueo es corto a propósito.
- **Cambio de contraseña con sesión abierta**: la identificación ya emitida sigue valiendo (fuera de alcance).
- **Integrante identificado que deja de pertenecer al proyecto**: se limpia como hoy en HU-20b.
- **"Salir" sin estar identificado**: responde igual (303 y "Cerraste tu identificación"); la cookie de proyecto no se
  toca.
- **"Salir" con `volver_a` externo o malformado**: se ignora y se vuelve al inicio.
- **Proyecto sin integrantes**: se muestra el mensaje de HU-20b y no hay formulario.
- **Comando sin `-integrante`, con un valor no numérico o fuera de rango**: muestra cómo se usa y termina con código
  2, sin pedir la contraseña.
- **Comando con integrante inexistente**: "El integrante no existe", código 1, sin pedir la contraseña.
- **Comando sin `DATABASE_URL`**: el mismo mensaje que el servidor ("falta la variable de entorno obligatoria
  DATABASE_URL"), código 1.
- **Comando con la entrada cerrada antes de tiempo**: lo que falta se toma como vacío (contraseña vacía → "al menos 8
  caracteres"; confirmación vacía → "no coinciden").
- **Comando con una contraseña corta que además no coincide**: se informa primero el largo.
- **Comando sobre un integrante que ya tenía contraseña**: la reemplaza (es la única forma de cambiarla).

## Requisitos funcionales

- **FR-001**: La pantalla "Identificarme" MUST pedir integrante y contraseña. El campo MUST ser
  `type="password"` con `autocomplete="current-password"` y MUST NOT mostrar nunca un valor.
- **FR-002**: Con la contraseña correcta, el sistema MUST identificar al integrante con la cookie firmada de HU-26,
  mostrar "Te identificaste como <Nombre>", mostrar su nombre en la estructura común y volver al destino saneado.
- **FR-003**: Con una contraseña incorrecta, el sistema MUST responder 401 con "Integrante o contraseña incorrectos",
  MUST NOT identificar, MUST conservar el integrante elegido con el campo de contraseña vacío y MUST contar un fallo.
- **FR-004**: Con la contraseña vacía, el sistema MUST responder 422 con "Ingresá tu contraseña" junto al campo, sin
  verificar ni contar un fallo.
- **FR-005**: Si el integrante no tiene contraseña configurada, el sistema MUST responder 422 con "<Nombre> todavía no
  tiene contraseña configurada. Pedile al Scrum Master que la configure", sin contar un fallo.
- **FR-006**: Después de 5 fallos seguidos de un integrante, el sistema MUST bloquearlo 1 minuto desde el quinto
  fallo y MUST responder 429 con "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" sin verificar la
  contraseña. Un acierto MUST reiniciar el contador; al vencer el bloqueo, el contador MUST volver a cero.
- **FR-007**: El bloqueo MUST ser por integrante y seguro para uso concurrente. Vive en memoria y se pierde al
  reiniciar el servidor (limitación aceptada).
- **FR-008**: Un integrante ausente, inválido o que no pertenece al proyecto activo MUST rechazarse con el
  comportamiento actual de HU-20b, sin verificar la contraseña ni contar un fallo.
- **FR-009**: Un rechazo por contraseña (401, 422 o 429) MUST NOT cambiar la identificación que ya tenía el
  navegador.
- **FR-010**: `POST /contexto/salir` MUST borrar la cookie del integrante sin tocar la del proyecto, mostrar
  "Cerraste tu identificación" y responder 303 al inicio, o al destino saneado. El botón "Salir" MUST verse en la
  estructura común junto al nombre del integrante identificado y solo entonces.
- **FR-011**: Al cambiar de proyecto, la identificación MUST limpiarse (HU-20b), y volver a identificarse MUST pedir
  la contraseña.
- **FR-012**: Una acción que requiere integrante sin identificación MUST llevar a la pantalla "Identificarme", que
  ahora pide contraseña (HU-20b, sin cambios en `RequireContext`).
- **FR-013**: El sistema MUST guardar solo un valor derivado con sal aleatoria por contraseña, con el formato
  `pbkdf2-sha256$<iteraciones>$<sal>$<hash>`. Dos integrantes con la misma contraseña MUST tener valores guardados
  distintos.
- **FR-014**: La verificación MUST comparar en tiempo constante y MUST usar las iteraciones del valor guardado. Un
  valor guardado mal formado MUST dar la página 500 sin identificar.
- **FR-015**: La contraseña MUST NOT aparecer en el registro del servidor, en las páginas, en los mensajes, en los
  errores, en direcciones ni en la salida del comando.
- **FR-016**: Si falla la lectura de los datos del integrante, el sistema MUST mostrar la página de error de HU-20a
  (503 o 500) sin detalles internos, MUST NOT identificar y MUST NOT contar un fallo.
- **FR-017**: El comando `set-password` con `-integrante N` y `DATABASE_URL` MUST pedir la contraseña y la
  confirmación, validarlas (8 a 128 caracteres contados como runas, iguales), guardar el valor derivado y mostrar
  "Contraseña configurada para <Nombre>".
- **FR-018**: Ante una contraseña inválida, el comando MUST informar el motivo ("La contraseña debe tener al menos 8
  caracteres", "La contraseña no puede superar 128 caracteres" o "Las contraseñas no coinciden"), MUST NOT cambiar nada
  y MUST terminar con un código distinto de 0.
- **FR-019**: Si el integrante no existe, el comando MUST mostrar "El integrante no existe" sin pedir la contraseña y
  MUST NOT cambiar nada.
- **FR-020**: La migración MUST agregar `members.password_hash` (texto, admite nulos) y su reversión MUST quitarla; los
  integrantes existentes quedan sin contraseña.
- **FR-021**: Las pruebas existentes que se identifican por `/contexto/integrante` MUST seguir en verde enviando la
  contraseña. Solo se reemplazan las expectativas de HU-20b sobre "sin contraseña".

## Entidades clave

- **Integrante** (`members`): suma la contraseña guardada (`password_hash`, nula si no se configuró).
- **Credencial del integrante**: lo que lee la pantalla para un integrante del proyecto activo: si pertenece, su
  nombre y su valor guardado.
- **Bloqueo**: fallos seguidos y momento del bloqueo de cada integrante, en memoria.

## Criterios de éxito

- **SC-001**: Cada uno de los 12 escenarios tiene al menos una prueba que falló antes del código (evidencia RED) y
  pasa después.
- **SC-002**: En la prueba del escenario 9, 0 apariciones de las contraseñas de prueba en la base, en las páginas, en
  el registro del servidor y en la salida del comando.
- **SC-003**: En las pruebas con reloj simulado, el 100% de los intentos durante el minuto de bloqueo se rechaza y, a
  los 60 segundos del quinto fallo, la contraseña correcta identifica.
- **SC-004**: Identificarse con la contraseña correcta tarda menos de 1 segundo en la verificación manual (con las
  600.000 iteraciones reales).
- **SC-005**: Toda la suite (`go test -count=1 -p 1 ./...`) queda en verde; en las pruebas existentes solo cambian las
  expectativas "sin contraseña" y el envío de la contraseña.
- **SC-006**: La migración se aplica y se revierte sin errores sobre una base descartable.

## Supuestos y decisiones

- La identificación sigue siendo de sesión del navegador (HU-20b) y la cookie sigue firmada (HU-26); HU-27 solo
  agrega la condición de la contraseña para emitirla.
- El Scrum Master conoce el número del integrante (por ejemplo, con
  `SELECT id, display_name FROM members`); no hay pantalla para verlo.
- Para la demo hay que configurar la contraseña de cada integrante después de migrar; si no, nadie puede
  identificarse (nota 7 del equipo).
- El reemplazo de "sin contraseña" afecta a HU-20b (escenario US4-1, FR-015 y su supuesto) y a
  `docs/hu-20b-deuda-tecnica.md` (decisión 1).
- El Issue no da contraseña para Dani ni dos integrantes con la misma contraseña; ver "Datos de ejemplo".
- Las respuestas 401 no llevan la cabecera `WWW-Authenticate`: no es autenticación HTTP sino un formulario.
- "Planificación de proyecto.pdf" no está en el repositorio; si dice otra cosa, prevalece y se corrige esta
  especificación antes de T002.

## Trazabilidad

| Escenario | Requisitos | Éxito | Prueba BDD (`test/`) | Pruebas unitarias, de handler e integración | Tareas |
|---|---|---|---|---|---|
| 1 Identificación correcta | FR-001, FR-002 | SC-001, SC-004 | `TestHU27_IdentificacionCorrecta` | `TestMemberPassword_Correcta`, `TestVerifyPassword_CorrectaEIncorrecta` | T005, T023, T024 |
| 2 Contraseña incorrecta | FR-003, FR-009 | SC-001 | `TestHU27_ContrasenaIncorrecta` | `TestMemberPassword_Incorrecta`, `TestMemberPassword_RechazoConservaIdentificacionPrevia` | T023, T024 |
| 3 Contraseña vacía | FR-004 | SC-001 | `TestHU27_ContrasenaVacia` | `TestMemberPassword_Vacia` | T023, T024 |
| 4 Sin contraseña | FR-005 | SC-001 | `TestHU27_IntegranteSinContrasena` | `TestMemberPassword_SinConfigurar` | T023, T024 |
| 5 Demasiados intentos | FR-006, FR-007 | SC-001, SC-003 | `TestHU27_DemasiadosIntentos` | `TestLockout_CincoFallosBloqueanUnMinuto`, `TestLockout_AciertoReiniciaElContador`, `TestLockout_PorIntegrante`, `TestLockout_UsoConcurrente`, `TestMemberPassword_Bloqueo` | T006, T023, T024 |
| 6 Otro proyecto | FR-008 | SC-001 | `TestHU27_IntegranteDeOtroProyecto` | `TestPostgreSQLContextStore_MemberCredential` | T012, T024 |
| 7 Salir | FR-010, FR-012 | SC-001 | `TestHU27_Salir` | `TestMemberPassword_Salir` | T023, T024 |
| 8 Cambio de proyecto | FR-011 | SC-001 | `TestHU27_CambioDeProyecto` | — | T024 |
| 9 Contraseña protegida | FR-013, FR-014, FR-015 | SC-001, SC-002 | `TestHU27_ContrasenaProtegida` | `TestHashPassword_FormatoYSalDistinta`, `TestHashPassword_IteracionesReales`, `TestVerifyPassword_LeeLasIteracionesGuardadas`, `TestMemberPassword_NuncaMuestraLaContrasena` | T005, T023, T024, T033 |
| 10 Configurar | FR-017 | SC-001 | `TestHU27_ConfigurarContrasena` | `TestSetPassword_Configura`, `TestRun_ConfiguraContrasena` (integración) | T019, T024 |
| 11 Inválida al configurar | FR-018 | SC-001 | — | `TestValidateNewPassword`, `TestSetPassword_ContrasenaInvalida` | T005, T019 |
| 12 Error al verificar | FR-014, FR-016 | SC-001 | `TestHU27_ErrorAlVerificar` | `TestMemberPassword_ErrorDeLectura`, `TestVerifyPassword_ValorGuardadoMalFormado`, `TestPostgreSQLContextStore_MemberCredentialBaseCaida` | T005, T012, T023, T024 |
| Comando: integrante inexistente y uso | FR-019 | SC-001 | — | `TestSetPassword_IntegranteInexistente`, `TestSetPassword_ErrorAlGuardar`, `TestRun_UsoYEntorno`, `TestPostgreSQLPasswordStore` | T013, T019 |
| Migración | FR-020 | SC-006 | — | `TestMigracion_PasswordHash_UpYDown` | T011 |
| Regresión de pruebas existentes | FR-021 | SC-005 | las 8 actualizadas (T001) | las 8 actualizadas (T001) | T029–T032 |
