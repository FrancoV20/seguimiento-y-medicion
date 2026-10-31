# Especificación: HU-19 – Resolver y reabrir defectos

**Rama**: HU-19
**Estado**: Implementada (pendiente de revisión en el Pull Request). Aprobada por el equipo el 22/10/2026 (sin Sprint activo se bloquea)
**Constitución**: v1.1.0
**Depende de**: HU-06, HU-04 / FIX-HU-04, HU-20a, HU-20b
**Habilita**: HU-10, HU-08, HU-09

## Historia de usuario

Como Product Builder, quiero cambiar el estado de un defecto y registrar el
Sprint en que se resolvió, para darle seguimiento hasta su cierre.

## Sección SDD

### Intención

Cerrar el ciclo de vida de un defecto: pasa de "Abierto" a "Resuelto"
registrando el Sprint en que se resolvió y puede volver a "Abierto" si
reaparece. El Sprint de resolución es el dato que HU-10 usa para contar los
defectos resueltos por Sprint.

### Alcance

- Acciones "Resolver" y "Reabrir" en la pantalla `/defectos` de HU-06.
- Reglas: Sprint activo obligatorio para resolver, orden entre Sprint de
  detección y de resolución, estados ya alcanzados y defecto inexistente.
- Cambio atómico del estado y del Sprint de resolución.

### Fuera de alcance

- Registrar defectos y mostrar la lista (HU-06).
- Editar descripción, severidad o historia de un defecto.
- Elegir manualmente el Sprint de resolución (siempre es el Sprint activo).
- Métricas de defectos por Sprint (HU-10) y su visualización (HU-08, HU-09).
- Historial de cambios de estado.
- Autenticación y roles (limitación conocida del proyecto).

### Contratos

- Rutas: `POST /defectos/resolver` y `POST /defectos/reabrir`, con el campo de
  formulario `defecto` (identificador). La pantalla sigue siendo `GET /defectos`.
- La sección es la de HU-06 (`RequiresProject: true`, sin integrante); el proyecto
  sale siempre del contexto del servidor.
- Códigos: 303 (éxito, redirige a `/defectos` con mensaje de una sola vista),
  404 (defecto inexistente), 409 (conflicto de estado o de regla) y 500 (falla al
  guardar). En 404, 409 y 500 se muestra otra vez `/defectos` con el mensaje de
  error y sin cambios en los datos.
- Mensajes definidos en un único archivo (ver `plan.md`).
- Protección contra envíos de otro origen de HU-20b (middleware global).

## Escenarios BDD

Los 9 escenarios del Issue HU-19 (texto idéntico):

1. Resolución exitosa.
2. Resolver sin Sprint activo.
3. Reabrir un defecto.
4. Sprint de resolución anterior al de detección.
5. Defecto inexistente.
6. Error al guardar.
7. Resolver un defecto ya resuelto.
8. Reabrir un defecto ya abierto.
9. La lista muestra la acción y el Sprint de resolución.

## Requisitos funcionales

- **FR-001**: La pantalla `/defectos` MUST mostrar, por cada defecto, su estado, su
  Sprint de detección y, si está resuelto, su Sprint de resolución (dato ya
  provisto por HU-06).
- **FR-002**: Un defecto "Abierto" MUST ofrecer la acción "Resolver" y un defecto
  "Resuelto" MUST ofrecer "Reabrir"; nunca ambas a la vez.
- **FR-003**: Resolver MUST pasar el defecto de "Abierto" a "Resuelto" y registrar
  el Sprint activo del proyecto como Sprint de resolución, en una sola operación
  atómica.
- **FR-004**: Si el proyecto no tiene Sprint activo, resolver MUST bloquearse con
  409 y el mensaje "No hay un Sprint activo para registrar la resolución del
  defecto", sin modificar el defecto.
- **FR-005**: Si el número del Sprint activo es menor que el del Sprint de
  detección, resolver MUST rechazarse con 409 y el mensaje "El Sprint de
  resolución no puede ser anterior al Sprint en que se detectó el defecto", sin
  modificar el defecto.
- **FR-006**: Resolver un defecto que ya está "Resuelto" MUST responder 409 con
  "El defecto ya está resuelto" y conservar su Sprint de resolución original.
- **FR-007**: Reabrir MUST pasar el defecto de "Resuelto" a "Abierto" y borrar su
  Sprint de resolución, en una sola operación atómica. No requiere Sprint activo.
- **FR-008**: Reabrir un defecto que ya está "Abierto" MUST responder 409 con "El
  defecto ya está abierto", sin modificar el defecto.
- **FR-009**: Si el identificador del defecto está ausente, no es un entero válido,
  no existe o pertenece a otro proyecto, el sistema MUST responder 404 con "El
  defecto no existe" y no modificar ningún defecto.
- **FR-010**: Si falla el guardado, el sistema MUST responder 500 con
  `MsgOperationFailed` ("La operación no pudo completarse") y el defecto MUST
  conservar su estado y su Sprint de resolución anteriores.
- **FR-011**: Ante un éxito, el sistema MUST responder 303 a `/defectos` y mostrar
  una sola vez "Defecto resuelto en el Sprint N" o "Defecto reabierto".
- **FR-012**: Resolver y reabrir MUST bloquear la fila del defecto durante la
  operación; resolver MUST además tomar el Sprint activo con bloqueo compartido,
  de modo que un cierre de Sprint (HU-15) concurrente se serialice con la
  resolución.
- **FR-013**: Resolver y reabrir MUST modificar únicamente el estado y el Sprint de
  resolución; descripción, severidad, historia y Sprint de detección MUST quedar
  intactos.
- **FR-014**: Las acciones MUST operar solo sobre defectos del proyecto activo; el
  proyecto MUST salir del contexto del servidor y no del formulario.
- **FR-015**: Un defecto "Resuelto" MUST tener siempre Sprint de resolución y uno
  "Abierto" MUST NOT tenerlo (también lo garantiza el CHECK de HU-06).
- **FR-016**: Las respuestas de error MUST NOT incluir texto de consultas ni
  detalles internos.

## Entidades clave

- **Defecto** (tabla `defects` de HU-06): estado "Abierto" o "Resuelto", Sprint de
  detección (obligatorio) y Sprint de resolución (solo si está resuelto).
- **Sprint**: número y estado ("Activo" o "Finalizado"); un solo activo por
  proyecto (FIX-HU-04).

## Criterios de éxito

- **SC-001**: En todos los casos de prueba, resolver deja el defecto "Resuelto" con
  el Sprint activo como Sprint de resolución.
- **SC-002**: En todos los casos de prueba, reabrir deja el defecto "Abierto" y sin
  Sprint de resolución.
- **SC-003**: Sin Sprint activo, ningún intento de resolver modifica datos.
- **SC-004**: Con Sprint activo anterior al de detección, ningún intento de
  resolver modifica datos.
- **SC-005**: Un identificador inexistente, inválido o de otro proyecto nunca
  modifica datos y responde 404.
- **SC-006**: Un fallo de guardado no deja cambios parciales.
- **SC-007**: Dos resoluciones simultáneas del mismo defecto terminan con una
  exitosa y otra con 409.
- **SC-008**: Cada escenario tiene una prueba que falló antes del código (evidencia
  RED) y pasa después (evidencia GREEN), y la cobertura de `src/defect` queda en
  `cobertura.txt` (al menos 80%, igual que HU-06). Excepción documentada: la parte de acciones de
  la lista del escenario 9 se implementó en T035 y T044, antes de escribir T048, que por eso es una
  prueba de caracterización (pasa en la evidencia RED del Bloque C).

## Supuestos y decisiones

- El Sprint de resolución es siempre el Sprint activo (no se elige).
- Reabrir un defecto de un Sprint finalizado está permitido y altera
  retroactivamente el conteo de HU-10 (documentado en las notas del Issue).
- Los números de Sprint de un proyecto crecen con el tiempo; el escenario 4 es una
  regla defensiva, probada con datos preparados en la base.
- Se reutiliza `ErrNoActiveSprint` de HU-06; el mensaje lo decide cada acción.

## Trazabilidad

| Historia | Escenario | Requisitos | Éxito | Prueba BDD / de handler | Prueba de dominio / integración |
|---|---|---|---|---|---|
| HU-19 | 1 Resolución exitosa | FR-003, FR-011, FR-013, FR-014, FR-015 | SC-001 | `TestHU19_ResolucionExitosa`, `TestEscenarioHU19_ResolucionExitosa` | `TestCheckResolve_OK`, `TestResolutionService_ResolveDevuelveElSprint`, `TestPostgresResolve_OK` |
| HU-19 | 2 Sin Sprint activo | FR-004 | SC-003 | `TestHU19_ResolverSinSprintActivo`, `TestEscenarioHU19_ResolverSinSprintActivo` | `TestCheckResolve_SinSprint`, `TestPostgresResolve_SinSprintActivo`, `TestPostgresResolve_SprintActivoDeOtroProyecto` |
| HU-19 | 3 Reabrir | FR-007, FR-011, FR-015 | SC-002 | `TestHU19_Reabrir`, `TestEscenarioHU19_ReabrirUnDefecto` | `TestCheckReopen_OK`, `TestResolutionService_ReopenDelegaEnElAlmacen`, `TestPostgresReopen_OK` |
| HU-19 | 4 Sprint anterior | FR-005 | SC-004 | `TestHU19_SprintAnteriorAlDeDeteccion` | `TestCheckResolve_Anterior`, `TestPostgresResolve_Anterior` |
| HU-19 | 5 Inexistente | FR-009, FR-014 | SC-005 | `TestHU19_DefectoInexistente`, `TestEscenarioHU19_DefectoInexistente` | `TestResolutionService_IdentificadoresInvalidos`, `TestPostgresResolve_Inexistente`, `TestPostgresResolve_OtroProyecto` |
| HU-19 | 6 Error al guardar | FR-010, FR-016 | SC-006 | `TestHU19_ErrorAlGuardar` | `TestResolutionService_PropagaLosErrores`, `TestPostgresResolve_FalloNoDejaCambios` |
| HU-19 | 7 Ya resuelto | FR-006 | SC-001 | `TestHU19_YaResuelto` | `TestCheckResolve_YaResuelto`, `TestPostgresResolve_YaResuelto` |
| HU-19 | 8 Ya abierto | FR-008 | SC-002 | `TestHU19_YaAbierto` | `TestCheckReopen_YaAbierto`, `TestPostgresReopen_YaAbierto` |
| HU-19 | 9 Lista con acción | FR-001, FR-002 | SC-001, SC-002 | `TestHU19_ListaMuestraAccionYSprint` (caracterización: pasa desde el rojo porque T035 y T044 ya agregaron las acciones) | — |
| HU-19 | Concurrencia | FR-012 | SC-007 | — | `TestPostgresResolve_Concurrente` |