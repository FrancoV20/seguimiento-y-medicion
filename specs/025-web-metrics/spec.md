# Feature Specification: Interfaz web de métricas del Sprint

**Feature Branch**: `HU-25`

**Created**: 2026-10-23

**Status**: Implementada (pendiente de integración a `main`)

**Input**: User description: "HU-25 / Interfaz Web de Métricas del Sprint (presenta HU-07): Como Scrum Master, quiero ver la velocidad y la desviación de esfuerzo de un Sprint en una pantalla, para evaluar la salud del proyecto. Depende de HU-07, HU-15, HU-17, HU-20a y HU-20b."

> **Alcance**: HU-25 presenta HU-07 en el navegador: una pantalla de solo lectura "Métricas del Sprint" que
> lista los Sprints finalizados del proyecto y muestra la velocidad y la desviación de esfuerzo del elegido.
> Los cálculos son de HU-07 y no se repiten en la capa web. No incluye Story Points planificados ni porcentaje
> de historias completadas (HU-10), detalle de historias (HU-16), gráficos (HU-08) ni exportar (HU-09).
>
> **Cierre bloqueado**: el escenario original "Cierre bloqueado" se quitó. La lista de historias a corregir se
> muestra en la pantalla "Cierre de Sprint" (HU-15). Esta pantalla solo muestra Sprints ya cerrados, que no
> reciben horas nuevas (HU-03), ni cambios de estimación (HU-17), ni ediciones (HU-13).
>
> **Sprints finalizados**: las métricas se muestran solo para Sprints en estado "Finalizado". El Sprint activo
> no figura; sus métricas se calculan al cerrarlo (HU-15).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ver las métricas de un Sprint finalizado (Priority: P1)

Como Scrum Master, quiero ver la velocidad y la desviación de esfuerzo de un Sprint finalizado, para evaluar si
el equipo cumplió lo planificado.

**Why this priority**: Es el objetivo de la historia y la parte visible del MVP: sin esta pantalla las métricas
de HU-07 no se pueden consultar desde la web.

**Independent Test**: Con un Sprint cerrado con historias "Hecho", horas estimadas y horas reales conocidas, abrir
la pantalla y verificar la velocidad y la desviación; repetir sin horas estimadas, con datos incompletos y sin
historias completadas.

**Acceptance Scenarios**:

1. **Given** un Sprint finalizado, **When** abro sus métricas, **Then** veo la velocidad y el porcentaje de
   desviación.
2. **Given** un Sprint sin horas estimadas, **When** abro sus métricas, **Then** veo "No disponible" en lugar de un
   error o un valor inválido, y la velocidad igual se muestra.
3. **Given** un Sprint con datos incompletos, **When** abro sus métricas, **Then** veo la alerta "Cálculo parcial:
   basado en X de Y historias".
4. **Given** un Sprint finalizado sin ninguna historia completada, **When** abro sus métricas, **Then** veo la
   velocidad en cero y la desviación como "No disponible", sin error.

---

### User Story 2 - Elegir el Sprint y casos vacíos (Priority: P1)

Como Scrum Master, quiero ver la lista de Sprints finalizados y elegir cuál consultar, y entender qué pasa cuando
no hay ninguno, para revisar el desempeño de cada iteración.

**Why this priority**: Sin elegir el Sprint solo se vería uno, y sin los casos vacíos la pantalla parecería rota
al empezar el proyecto.

**Independent Test**: Con tres Sprints finalizados y uno activo, abrir la pantalla y elegir cada Sprint; con un
proyecto sin Sprints finalizados, verificar el mensaje.

**Acceptance Scenarios**:

1. **Given** un proyecto con varios Sprints finalizados, **When** abro "Métricas del Sprint", **Then** veo la lista
   de Sprints finalizados del más reciente al más antiguo, con las métricas del más reciente, y al elegir otro veo
   las suyas.
2. **Given** un proyecto sin Sprints finalizados, **When** abro "Métricas del Sprint", **Then** veo "Aún no hay
   Sprints finalizados".
3. **Given** un proyecto con un Sprint activo y Sprints finalizados, **When** abro "Métricas del Sprint", **Then**
   el Sprint activo no figura en la lista y veo que sus métricas se calculan al cerrarlo.

---

### Edge Cases

- Un `sprint` no numérico, inexistente, activo o de otro proyecto en la dirección se ignora: se muestra el Sprint
  finalizado más reciente (o el mensaje de que no hay).
- Un Sprint cerrado sin historias o sin ninguna en "Hecho" muestra velocidad cero y desviación "No disponible".
- Las historias devueltas al Backlog al cerrar (HU-15) no cuentan: HU-07 trabaja con las "Hecho" del Sprint.
- Un Sprint con una sola historia con horas estimadas de varias terminadas muestra el cálculo parcial con X de Y.
- La desviación puede ser positiva (se trabajó más de lo estimado), negativa (menos) o cero.
- Si el Sprint activo existe pero no hay Sprints finalizados, se muestra el mensaje de "Aún no hay Sprints
  finalizados" junto con el aviso del Sprint en curso.
- Un Sprint Goal con HTML o scripts se muestra como texto (regla de HU-20b).
- La pantalla es de solo lectura: no tiene formularios y un `POST` a `/metricas` recibe la página 405 de HU-20a.
- Sin proyecto activo válido, HU-20b redirige a la selección de proyecto.
- Si falla la lectura de los Sprints o el cálculo de HU-07, se muestra la página de error de HU-20a y no métricas
  incompletas.
- Las métricas se calculan al abrir la pantalla; no se guardan en esta historia.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: El sistema MUST ofrecer la sección "Métricas del Sprint" en el menú, que requiere un proyecto
  activo.
- **FR-002**: La pantalla MUST listar los Sprints finalizados del proyecto activo, del más reciente al más antiguo,
  con número, Sprint Goal y fechas.
- **FR-003**: La pantalla MUST mostrar la velocidad (Story Points) y la desviación de esfuerzo del Sprint elegido,
  por defecto el finalizado más reciente, tal como las calcula HU-07, sin recalcularlas.
- **FR-004**: La desviación MUST mostrarse con dos decimales, coma y signo (`+20,00 %`, `-8,50 %`, `0,00 %`), junto
  con la fórmula (horas reales − horas estimadas) ÷ horas estimadas.
- **FR-005**: Si el Sprint no tiene horas estimadas, la desviación MUST mostrarse como "No disponible" y la
  velocidad MUST mostrarse igual, sin error ni valor inválido.
- **FR-006**: Con datos incompletos la pantalla MUST mostrar la alerta "Cálculo parcial: basado en X de Y
  historias" con los valores de HU-07.
- **FR-007**: Un Sprint sin historias completadas MUST mostrar velocidad cero y desviación "No disponible".
- **FR-008**: El Sprint activo MUST NOT figurar en la lista ni calcularse; la pantalla MUST indicar que sus métricas
  se calculan al cerrarlo.
- **FR-009**: Sin Sprints finalizados la pantalla MUST mostrar "Aún no hay Sprints finalizados".
- **FR-010**: Un `sprint` inválido, activo o de otro proyecto en la dirección MUST ignorarse.
- **FR-011**: Ante una falla de lectura o de cálculo el sistema MUST mostrar la página de error de la aplicación y
  MUST NOT mostrar métricas incompletas.
- **FR-012**: Los mensajes de error MUST NOT exponer detalles internos de la base de datos.
- **FR-013**: Los textos ingresados (Sprint Goal) MUST mostrarse escapados como texto.
- **FR-014**: El proyecto MUST salir del contexto del servidor, todas las lecturas MUST filtrarse por él y la
  pantalla MUST ser de solo lectura.

### Especificación SDD *(requerida por el DoD)*

**Objetivo**: presentar en una pantalla web la velocidad y la desviación de esfuerzo que calcula HU-07 para los
Sprints finalizados de un proyecto, mostrando el cálculo parcial y la falta de datos de forma clara y sin errores.

**Entradas**:

| Solicitud | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `GET /metricas` | `sprint` (opcional) | identificador numérico | no | Elige el Sprint; si falta o es inválido se usa el finalizado más reciente |

El proyecto sale del contexto del servidor. No hay formularios ni `POST`.

**Salidas esperadas**:

- Con Sprints finalizados: página 200 con la lista, el Sprint elegido, la velocidad, la desviación y, si
  corresponde, la alerta de cálculo parcial.
- Sin Sprints finalizados: página 200 con el mensaje y, si hay un Sprint activo, el aviso de que sus métricas se
  calculan al cerrarlo.
- Falla de lectura o cálculo: página de error de HU-20a.
- Sin proyecto activo válido: redirección de HU-20b.

**Reglas de negocio**:

- Solo se muestran Sprints en estado "Finalizado" del proyecto activo.
- La velocidad y la desviación son las de HU-07; la web no las recalcula.
- La desviación se muestra con dos decimales, coma y signo; sin horas estimadas se muestra "No disponible".
- Un cálculo parcial siempre se acompaña de la alerta con X de Y.

**Restricciones**:

- Sin migraciones ni dependencias nuevas. No se modifican archivos de HU-07, HU-15, HU-17, HU-20a ni HU-20b.
- Ningún mensaje expone consultas, rutas ni valores de configuración.
- La pantalla funciona sin JavaScript y sin conexión a internet (HU-20a).

**Casos límite**: ver Edge Cases.

**Condiciones de error**:

| Condición | Mensaje | Respuesta |
|---|---|---|
| Sin Sprints finalizados | "Aún no hay Sprints finalizados" | 200 |
| Sin horas estimadas | "No disponible" (HU-07) | 200 |
| Cálculo parcial | "Cálculo parcial: basado en X de Y historias" (HU-07) | 200 |
| `sprint` inválido, activo o ajeno | Se ignora | 200 |
| Falla de lectura o de cálculo | Página de error de HU-20a | 500 o 503 |
| Sin proyecto activo válido | Mensaje de HU-20b | 303 |
| `POST` a `/metricas` | Página 405 de HU-20a | 405 |

Los textos propios de HU-25 son una propuesta del equipo y se definen en un solo lugar (`metrics_view.go`).

### Key Entities *(include if feature involves data)*

- **Sprint finalizado**: Iteración cerrada del proyecto (número, Sprint Goal, fechas).
- **Métricas del Sprint**: Velocidad (Story Points), desviación de esfuerzo y, si corresponde, X de Y historias del
  cálculo parcial, tal como las devuelve HU-07.
- **Selección de Sprint**: El Sprint cuyas métricas se muestran, elegido por la dirección o el más reciente.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: El 100% de las métricas mostradas coincide con las que HU-07 calcula para el mismo Sprint.
- **SC-002**: Ningún Sprint activo ni de otro proyecto aparece en la lista ni se calcula.
- **SC-003**: Ningún caso sin horas estimadas muestra un error ni un valor inválido: siempre "No disponible".
- **SC-004**: Todo cálculo parcial muestra la alerta con X de Y.
- **SC-005**: La pantalla responde en menos de 500 ms bajo condiciones normales (la prueba de T012 lo mide con la
  base local, hasta 20 Sprints finalizados y un Sprint con menos de 50 historias).
- **SC-006**: Ninguna acción de la pantalla modifica datos.

## Assumptions

- HU-07 expone un servicio que, dado un Sprint, devuelve la velocidad, la desviación (o su falta) y los valores X e
  Y del cálculo parcial; HU-25 lo consume (se verifica en T001).
- HU-15 deja el Sprint en estado "Finalizado" y las historias "Hecho" asignadas a él; las devueltas ya no cuentan.
- HU-17 y HU-24 (o la carga de HU-03) son las fuentes de las horas estimadas y reales que HU-07 usa.
- No hay roles ni identificación para consultar las métricas; la autenticación y los roles se resuelven después del
  Sprint 2 (decisión del equipo).
- Estructura común, errores y página 405 vienen de HU-20a; el proyecto activo y la redirección, de HU-20b.

## Trazabilidad *(Historia → Spec → Criterios → Escenarios BDD → Tests → Código)*

| Escenario del Issue | Requisitos | Tests | BDD | Código |
|---|---|---|---|---|
| 1. Métricas de un Sprint finalizado | FR-003, FR-004 | T008, T010, T012 | T016 | `metrics_handlers.go`, `metrics_source.go`, `metrics_view.go`, `metrics.html` |
| 2. Sin esfuerzo estimado | FR-005 | T008, T010, T013 | T016 | `metrics_source.go`, `metrics_view.go` |
| 3. Cálculo parcial | FR-006 | T008, T010, T014 | T016 | `metrics_source.go`, `metrics_view.go`, `metrics.html` |
| 4. Velocidad cero | FR-007 | T008, T015 | T016 | `metrics_source.go` |
| 5. Elegir otro Sprint | FR-001, FR-002, FR-010 | T006, T010, T020, T021 | T024 | `postgres_metrics_catalog.go`, `metrics_view.go`, `metrics_handlers.go`, `metrics.html` |
| 6. Sin Sprints finalizados | FR-009 | T010, T022 | T024 | `metrics_view.go`, `metrics_handlers.go`, `metrics.html` |
| 7. Sprint en curso | FR-008 | T006, T010, T023 | T024 | `postgres_metrics_catalog.go`, `metrics_view.go`, `metrics.html` |
| (HU-20b) Sin proyecto activo | FR-001, FR-014 | T033 | | `metrics_handlers.go` (`RequiresProject`) |
| (HU-20b) HTML como texto | FR-013 | T034 | | `metrics.html` |
| (HU-20a) Error al leer o calcular | FR-011, FR-012 | T031, T032 | | `metrics_handlers.go` (`Fail`) |
| (HU-20a) POST no soportado | FR-014, SC-006 | T035 | | `metrics_handlers.go` (solo `GET`), registrada en `src/cmd/server/main.go` |

Trazabilidad revisada en T039 (31/10/2026); el detalle por requisito y criterio, con los nombres de las pruebas, está en
`tasks.md`.