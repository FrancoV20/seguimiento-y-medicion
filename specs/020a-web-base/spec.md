# Feature Specification: Base de la aplicación web

**Feature Branch**: `020a-web-base`

**Created**: 2026-10-17 | **Updated**: 2026-10-18

**Status**: Draft

**Input**: User description: "HU-20a / Arranque, Estructura Común y Manejo de Errores: Como Usuario, quiero abrir la aplicación en el navegador y ver la misma estructura y mensajes claros en todas las pantallas, y como equipo de desarrollo quiero que el servidor falle o se recupere de forma clara ante problemas de configuración, de base de datos o de apagado, para usar y diagnosticar el sistema sin comandos ni exponer datos sensibles."

> **División de HU-20**: HU-20a construye el servidor, la estructura común y el manejo de errores. HU-20b
> construye los formularios y mensajes, la protección contra envíos ajenos y el contexto de trabajo
> (proyecto e integrante activos). Esta historia no depende de ninguna otra.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Arrancar la aplicación y ver la página de inicio (Priority: P1)

Como Usuario, quiero abrir la dirección de la aplicación en el navegador y ver una página de
inicio con el menú de navegación, para usar el sistema sin comandos. Como integrante del equipo
que levanta el servidor, quiero que los problemas de arranque se informen con mensajes claros.

**Why this priority**: Sin un servidor que arranque de forma confiable y una página de inicio,
ninguna otra historia web (HU-20b, HU-21 a HU-25) puede entregarse ni demostrarse.

**Independent Test**: Iniciar el servidor con la base de datos disponible y abrir su dirección;
luego repetir el arranque con configuración ausente, base de datos lenta, base de datos caída,
migraciones pendientes, esquema en estado "dirty" y puerto ocupado.

**Acceptance Scenarios**:

1. **Given** que la configuración es válida, la base de datos responde y el esquema está al día,
   **When** se inicia el servidor y abro su dirección en el navegador, **Then** veo la página de
   inicio con el menú de navegación.
2. **Given** que falta la variable `DATABASE_URL`, **When** se inicia el servidor, **Then**
   informa qué variable falta, sin mostrar valores sensibles, y no queda en ejecución.
3. **Given** que la base de datos todavía no está lista, **When** se inicia el servidor, **Then**
   reintenta la conexión hasta 10 veces con 2 segundos de espera y, si conecta, continúa con el
   arranque normal.
4. **Given** que la base de datos sigue sin responder después de los reintentos, **When** se
   inicia el servidor, **Then** informa que no pudo conectarse, sin mostrar la clave ni la URL de
   conexión completa, y no queda en ejecución.
5. **Given** que la versión del esquema es menor que la última migración disponible, **When** se
   inicia el servidor, **Then** no queda en ejecución e informa que el esquema está desactualizado
   y qué comando ejecutar para aplicar las migraciones.
6. **Given** que el esquema quedó en estado "dirty", **When** se inicia el servidor, **Then** no
   queda en ejecución e indica avisar al equipo antes de forzar la versión.
7. **Given** que el puerto configurado ya está en uso o no es válido, **When** se inicia el
   servidor, **Then** informa cuál es el problema y qué puerto está afectado, y no queda en
   ejecución.
8. **Given** que el servidor recibe una orden de detención, **When** hay solicitudes en curso,
   **Then** deja de aceptar solicitudes nuevas, completa las que están en curso durante un máximo
   de 10 segundos y se cierra liberando la conexión a la base de datos.

---

### User Story 2 - Navegar entre secciones con una estructura consistente (Priority: P1)

Como Usuario, quiero que todas las pantallas tengan el mismo menú y la misma estructura, para
orientarme y moverme entre secciones sin perderme.

**Why this priority**: HU-20b y HU-21 a HU-25 agregan pantallas dentro de esta estructura;
definirla una sola vez evita decisiones distintas en cada historia.

**Independent Test**: Registrar secciones de prueba y verificar el menú, la sección resaltada, el
espacio de contexto, la ausencia de recursos externos y el texto con caracteres especiales.

**Acceptance Scenarios**:

1. **Given** cualquier página, **When** se muestra, **Then** incluye el menú de navegación, la
   sección actual resaltada y un espacio de contexto que muestra "Sin proyecto seleccionado"
   cuando no hay contexto activo.
2. **Given** que hay secciones registradas y otras todavía no implementadas, **When** se muestra
   el menú, **Then** aparecen solo las registradas, y una sección nueva aparece apenas su
   historia la registra, sin modificar la estructura común.
3. **Given** que no hay conexión a internet, **When** abro la aplicación, **Then** estilos,
   fuentes y scripts cargan igual, porque no dependen de servicios externos.
4. **Given** un texto con tildes, ñ y símbolos, **When** se muestra en cualquier pantalla,
   **Then** se ve correctamente.

---

### User Story 3 - Recibir errores claros sin exponer detalles internos (Priority: P2)

Como Usuario, quiero que cuando algo falla o no existe vea un mensaje claro y pueda volver al
inicio, sin ver rutas, consultas ni claves, para no quedar bloqueado ni conocer detalles internos.

**Why this priority**: Es un requisito de robustez que pesa en la evaluación de calidad y evita
que cada pantalla improvise su propio manejo de errores.

**Independent Test**: Solicitar direcciones inexistentes, parámetros malformados y métodos no
soportados; provocar un fallo inesperado; detener la base de datos con el servidor ya en
ejecución y volver a levantarla.

**Acceptance Scenarios**:

1. **Given** una dirección que no existe o con un parámetro no numérico o malformado (por
   ejemplo, `/algo/abc`), **When** la solicito, **Then** veo una página de error 404 con la
   estructura común y un enlace al inicio, sin rutas internas, consultas ni detalles de la base
   de datos, y nunca un error interno.
2. **Given** una dirección existente, **When** la solicito con un método no soportado, **Then**
   veo una página de error 405 con la estructura común y el servidor continúa en ejecución.
3. **Given** un fallo inesperado al procesar una pantalla, **When** ocurre, **Then** el servidor
   sigue en ejecución, veo una página de error 500 genérica con la estructura común y un enlace
   al inicio, y el detalle técnico queda solo en el registro del servidor.
4. **Given** que el servidor está en ejecución y la base de datos deja de responder, **When**
   solicito una página que necesita la base, **Then** veo una página 503 con el texto "El
   servicio no está disponible temporalmente", sin detalles internos, y la estructura común se
   muestra sin consultar la base de datos.
5. **Given** que la base de datos vuelve a responder, **When** solicito nuevamente una página,
   **Then** funciona con normalidad sin reiniciar el servidor.

---

### Edge Cases

- Los mensajes de error de configuración y de conexión nunca incluyen valores de variables ni
  claves; solo nombres de variables y descripciones.
- Si falla la construcción de una página, el usuario recibe la página de error completa y nunca
  una página a medias.
- Una dirección con un identificador numérico muy grande o negativo se trata igual que un
  identificador malformado.
- Si dos secciones intentan registrar la misma dirección, el servidor no inicia e informa el
  conflicto, porque es un error de programación y no de uso.
- Si la versión de esquema de la base de datos es más nueva que la esperada por el código, el
  servidor inicia y deja una advertencia en su registro.
- Si la base de datos cae, la estructura común (menú, estilos) se sigue mostrando: solo falla la
  parte de la página que necesita datos.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST mostrar la página de inicio con el menú de navegación al abrir la
  dirección del servidor.
- **FR-002**: Si falta una variable de configuración obligatoria, el sistema MUST informar su
  nombre sin mostrar valores y MUST NOT quedar en ejecución.
- **FR-003**: El sistema MUST reintentar la conexión a la base de datos (10 intentos, 2 segundos
  entre intentos) antes de considerarla no disponible.
- **FR-004**: Si no logra conectarse, el sistema MUST informarlo sin mostrar credenciales ni la
  URL de conexión completa y MUST NOT quedar en ejecución.
- **FR-005**: Al iniciar, el sistema MUST verificar el estado del esquema de la base de datos y,
  si hay migraciones pendientes o el esquema está en estado "dirty", MUST NOT quedar en
  ejecución e informar el motivo y la acción a tomar.
- **FR-006**: Si el puerto configurado está en uso o no es válido, el sistema MUST informar el
  problema y el puerto afectado, y MUST NOT quedar en ejecución.
- **FR-007**: El sistema MUST detenerse de forma ordenada: dejar de aceptar solicitudes nuevas,
  completar las en curso durante un máximo de 10 segundos y liberar la conexión a la base de
  datos.
- **FR-008**: Todas las páginas MUST compartir una estructura común con menú, sección actual
  resaltada y un espacio de contexto que muestra "Sin proyecto seleccionado" cuando no hay
  contexto activo.
- **FR-009**: El menú MUST mostrar solo las secciones registradas y MUST incorporar una nueva
  sección cuando esta se registra, sin cambios en la estructura común.
- **FR-010**: Las páginas MUST NOT depender de recursos alojados fuera del servidor.
- **FR-011**: Todos los textos MUST mostrarse correctamente con tildes, ñ y símbolos.
- **FR-012**: El sistema MUST responder con una página 404 (estructura común y enlace al inicio)
  para direcciones inexistentes y para parámetros de dirección malformados, nunca un error
  interno.
- **FR-013**: El sistema MUST responder con una página 405 (estructura común) a los métodos de
  solicitud no soportados y continuar en ejecución.
- **FR-014**: Ante un fallo inesperado, el sistema MUST mantenerse en ejecución, mostrar una
  página 500 genérica con la estructura común y un enlace al inicio, y registrar el detalle solo
  en el servidor.
- **FR-015**: Cuando la base de datos no responde con el servidor en ejecución, el sistema MUST
  mostrar la página 503 con "El servicio no está disponible temporalmente" sin consultar la base
  para la estructura común, y MUST recuperarse sin reinicio cuando la base vuelva.
- **FR-016**: Ninguna página de error MUST exponer rutas de archivos, consultas, trazas ni
  credenciales.
- **FR-017**: El sistema MUST NOT enviar páginas incompletas ante un fallo de construcción.

### Key Entities *(include if feature involves data)*

- **Sección**: Apartado del menú con nombre, dirección y requisitos de contexto (proyecto,
  integrante); se registra cuando su historia se implementa. HU-20b usa los requisitos para
  proteger secciones.
- **Estado del esquema**: Versión aplicada de la base de datos y si está en estado "dirty".

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de los arranques con configuración obligatoria ausente termina con un
  mensaje que nombra la variable faltante y no contiene valores de configuración.
- **SC-002**: Si la base de datos queda disponible dentro de la ventana de reintentos, el
  servidor inicia correctamente sin intervención manual.
- **SC-003**: El 100% de las direcciones inexistentes o con parámetros malformados responde
  "no encontrado" y 0 responden error interno.
- **SC-004**: 0 respuestas de error contienen rutas de archivos, texto de consultas, trazas o el
  valor de `DATABASE_URL`, verificado por pruebas automatizadas.
- **SC-005**: Tras recuperarse la base de datos, la primera solicitud posterior se responde
  normalmente, sin reiniciar el servidor.
- **SC-006**: Con la red desconectada, la página de inicio carga completa sin solicitudes a
  otros orígenes.
- **SC-007**: La página de inicio responde en menos de 500 ms bajo condiciones normales de uso.

## Assumptions

- La aplicación se entrega como páginas generadas por el servidor, sin autenticación de
  usuarios; el acceso se considera confiable dentro del equipo.
- El espacio de contexto es parte de la estructura común; esta historia lo deja con "Sin proyecto
  seleccionado" y HU-20b lo completa con el proyecto y el integrante activos.
- Las secciones de HU-21 a HU-25 se registran al implementarse; esta historia no incluye sus
  pantallas, solo la estructura que las aloja. Mientras no existan, las pruebas usan secciones
  de prueba.
- Los textos de la interfaz están en español.
- La dirección de escucha es la variable `ADDR` (por defecto `:8080`); los tiempos (10 reintentos
  de 2 segundos y apagado de 10 segundos) están fijados en `plan.md`.
- La versión esperada del esquema es el mayor número entre las migraciones `.up.sql` del
  repositorio.
- No se crean tablas ni migraciones; la verificación de esquema lee el control de versiones que
  ya mantiene la herramienta de migraciones.
- Mensajes de éxito, error y advertencia, formularios, protección de contenido y de envíos,
  proyecto e integrante activos, roles, inicio de sesión, idiomas adicionales y adaptación
  avanzada a móviles quedan fuera de esta historia (los tres primeros grupos son de HU-20b).
