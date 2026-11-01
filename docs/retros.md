# Actas del Equipo

Registro de reuniones de seguimiento (dailies informales), Sprint Reviews y Retrospectivas
del proyecto **Software Metrics & Estimation**.

---

## Sprint 1

**Sprint Goal:** Dejar funcionando el ciclo básico de gestión: crear un proyecto, cargarle
historias al backlog, organizarlas en un sprint, y que el sistema empiece a calcular
métricas de seguimiento.

**Historias:** HU-01, HU-02, HU-04, HU-07 (planificadas) y HU-03 (incorporada durante el Sprint)

**Fecha de cierre:** 18 de octubre de 2026

### Seguimiento (dailies informales)

Las dailies del Sprint 1 fueron informales y no se registraron por escrito. A partir del
Sprint 2 se anota una línea por sesión de trabajo (qué se hizo, qué sigue, bloqueos).

---

### Sprint Review

- **Fecha:** 18 de octubre de 2026
- **Historias completadas:**
  - HU-01 Alta de Nuevo Proyecto (3 SP)
  - HU-02 Creación de Historia en el Backlog (5 SP)
  - HU-03 Registro de horas trabajadas (3 SP, incorporada durante el Sprint)
  - HU-04 Creación y Planificación de Sprint (8 SP)
  - HU-07 Cálculo de Velocidad y Desviación de Esfuerzo (5 SP)
  - Total: 24 SP completados sobre 21 SP planificados al inicio.
- **Historias no completadas (y por qué):** ninguna de las comprometidas. Quedaron fuera de este
  Sprint las historias que se detectaron como faltantes al contrastar el backlog con el PDF de
  la cátedra (HU-11 a HU-25: cierre de Sprint, estado de historias, consulta de Sprints
  anteriores, horas estimadas, interfaz web, entre otras). Se planifican entre los Sprints 2 y 4.
- **Resultado frente al Sprint Goal:** el núcleo de reglas de negocio está implementado y probado
  contra PostgreSQL real (proyectos, backlog, esfuerzo, Sprints y métricas). El MVP completo
  no se alcanzó en este Sprint porque faltaban historias en el backlog; se alcanza con el
  Sprint 2.
- **Feedback del Product Architect:** (completar luego de la presentación)

---

### Retrospectiva

- **Fecha:** 18 de octubre de 2026
- **Qué funcionó bien:**
  - Especificar cada historia con SDD, BDD y TDD antes de programar dejó pruebas reproducibles
    y evidencia por historia.
  - Probar contra PostgreSQL real, y no contra datos simulados, detectó problemas que de otro
    modo se habrían visto recién al integrar.
  - La base de datos se decidió con una encuesta rápida del equipo y se mantuvo.
  - Contrastar el backlog con el PDF de la cátedra permitió detectar las historias faltantes
    antes de cerrar el Proyecto.
  - Reasignar HU-03 a la Scrum Master destrabó la dependencia de HU-07.
- **Qué no funcionó / mejorar:**
  - El backlog estaba incompleto respecto del PDF: el MVP no estaba cubierto y se detectó recién
    al cerrar el Sprint.
  - Las dependencias entre historias no se analizaron al planificar: HU-07 dependía de HU-03.
  - Las migraciones se numeraron por número de historia y se aplicaron fuera de orden:
    `migrate up` no aplica una migración de número menor a la versión ya aplicada, y la 003 se
    salteó en una base.
  - Copiar archivos de otras historias entre ramas para poder probar generó conflictos de
    merge en el PR de HU-07 (unos 20 archivos).
  - Una prueba de integración de HU-01 se rompió al existir tablas que referencian a sus
    tablas, y las pruebas de integración corridas en paralelo se pisan porque comparten una
    única base.
  - Documentación desactualizada: el `tasks.md` de HU-07 tenía la carpeta y los números de
    migración viejos, y el README tenía el orden de migraciones viejo.
  - El cierre del Sprint se movió del 9 al 18 de octubre.
- **Acciones para el próximo sprint:**
  1. Una migración nueva toma el último número de `main` más uno, sin importar el número de su
     historia, y se agrega después de que sus dependencias están en `main`.
  2. Mergear los PR con "Create a merge commit" y hacer `git merge origin/main` seguido en cada
     rama, en lugar de copiar archivos de otras historias.
  3. Correr las pruebas con `go test -count=1 -p 1 ./...` y dejarlo escrito en el README y en
     los quickstarts.
  4. Revisar la matriz de dependencias antes de cada planning y marcar las dependencias en cada
     Issue ("Depende de").
  5. Verificar con up, down y up desde cero la migración `down` de HU-04.
  6. Corregir en el README el orden y los dueños de las migraciones.
  7. Agregar un `.gitattributes` para unificar los saltos de línea y que `gofmt -l` no marque
     archivos por el formato de Windows.
  8. Registrar las dailies en este archivo, una línea por sesión.
  9. Hacer el Planning Poker del Sprint 2 incluyendo HU-20a y HU-20b y recalcular la capacidad.

---

## Sprint 2

**Sprint Goal:** Al cierre del Sprint 2, un integrante puede completar desde el navegador el ciclo
del MVP: crear un proyecto, cargar historias al Backlog, armar un Sprint, pasar historias a En
progreso y Hecho, registrar horas estimadas y reales, cerrar el Sprint y ver su velocidad y
desviación de esfuerzo.

**Fechas:** del 18 de octubre de 2026 (Sprint Planning y Planning Poker) al 1 de noviembre de 2026.

**Historias:** HU-20a, HU-20b, HU-21, HU-22, HU-23, HU-14, HU-15, HU-17, HU-24 y HU-25
(planificadas, 44 SP según el Planning Poker del 18/10), más FIX-HU-01 y FIX-HU-04 (correcciones
surgidas del Sprint 1, sin Story Points).

**Fecha de cierre:** 1 de noviembre de 2026.

### Seguimiento

Las dailies del Sprint 2 no se registraron por escrito (la acción 8 de la retro anterior no se
cumplió). El avance se reconstruye desde las fechas de integración a `main`:

| Fecha | Integrado a `main` |
|---|---|
| 19/10 y 20/10 | HU-20a (PR #22 y segunda versión en PR #24) |
| 21/10 | HU-20b (#28), FIX-HU-01 (#30) |
| 22/10 | FIX-HU-04 (#32) |
| 26/10 | HU-21 (#36), primera pantalla web de negocio |
| 27/10 | HU-22 (#38), HU-14 (#41), HU-24 (#37), pipeline de CI (#44) |
| 28/10 | HU-23 (#50) |
| 29/10 | HU-15 (#59) |
| 30/10 | HU-17 (#56) |
| 01/11 | HU-25 (#64), la última del Sprint |

---

### Sprint Review

- **Fecha:** 1 de noviembre de 2026
- **Historias completadas:**

  | Historia | SP | Responsable | PR |
  |---|---|---|---|
  | HU-20a Arranque, Estructura Común y Manejo de Errores | 8 | Alejandro | #22, #24 |
  | HU-20b Formularios Seguros y Contexto de Trabajo | 8 | Franco | #28 |
  | HU-21 Interfaz Web de Proyectos | 3 | Adriel | #36 |
  | HU-22 Interfaz Web del Product Backlog | 3 | Giuliana | #38 |
  | HU-23 Interfaz Web de Sprints | 5 | Ignacio | #50 |
  | HU-14 Actualización de Estado (Tablero) | 3 | Franco | #41 |
  | HU-15 Cierre de Sprint | 5 | Giuliana | #59 |
  | HU-17 Registro de Horas Estimadas | 3 | Ignacio | #56 |
  | HU-24 Interfaz Web de Registro de Esfuerzo | 3 | Alejandro | #37 |
  | HU-25 Interfaz Web de Métricas del Sprint | 3 | Adriel | #64 |
  | FIX-HU-01 Fecha de fin igual a la de inicio | – | Adriel | #30 |
  | FIX-HU-04 Un solo Sprint activo por proyecto | – | Alejandro | #32 |

  - Total: **44 SP completados sobre 44 planificados** (10 de 10 historias, 100,00 %).
  - Velocidad: **44 SP** (Sprint 1: 24 SP).
- **Historias no completadas (y por qué):** ninguna.
- **Además se entregó:** pipeline de integración continua en GitHub Actions (`gofmt`, `go vet` y
  `go test`), con una cobertura total de 85,3 % al 27/10.
- **Resultado frente al Sprint Goal:** cumplido. El ciclo completo del MVP se puede recorrer desde
  el navegador. Una salvedad: después de integrar HU-21 y HU-22 apareció un defecto (FIX-HU-21:
  crear una historia en un proyecto dado de alta desde la web devolvía un error 500), que se
  corrigió el 29/10 (#58). Desde ese día el ciclo funciona de punta a punta.
- **Trabajo adelantado:** a partir del 25/10, los integrantes que terminaban su historia del
  Sprint 2 tomaron historias del Sprint 3 (HU-05a, HU-05b, HU-06, HU-10, HU-11, HU-12, HU-16,
  HU-18, HU-19 y HU-26). No se cuentan en la velocidad del Sprint 2; se registran en el Sprint 3.
- **Feedback del Product Architect:** (completar luego de la presentación)

---

### Retrospectiva

- **Fecha:** 1 de noviembre de 2026
- **Acciones de la retro anterior:**

  | # | Acción | Estado |
  |---|---|---|
  | 1 | Numerar migraciones nuevas con el último número de `main` más uno | Cumplida (011 a 014 en orden) |
  | 2 | Merge commits y `git merge origin/main` seguido en cada rama | Cumplida |
  | 3 | `go test -count=1 -p 1 ./...` escrito en el README y los quickstarts | Pendiente en el README |
  | 4 | Matriz de dependencias y "Depende de" en cada Issue | Cumplida |
  | 5 | Verificar up, down y up de la migración de HU-04 | (confirmar) |
  | 6 | Orden y dueños de las migraciones en el README | (confirmar) |
  | 7 | Agregar `.gitattributes` | No cumplida |
  | 8 | Registrar las dailies, una línea por sesión | No cumplida |
  | 9 | Planning Poker del Sprint 2 con HU-20a y HU-20b | Cumplida (18/10) |

- **Qué funcionó bien:**
  - Se completó el 100 % de lo comprometido y la velocidad pasó de 24 a 44 SP.
  - El pipeline de CI detecta formato, `go vet` y pruebas rotas antes del merge.
  - Traer `main` seguido a cada rama evitó los conflictos masivos del Sprint 1.
  - Las especificaciones con "Depende de" y "Orden de implementación" permitieron trabajar varias
    historias en paralelo sin pisarse.
  - El sistema pull funcionó: quien terminaba su historia tomaba la siguiente sin esperar.
- **Qué no funcionó / mejorar:**
  - Cuello de botella inicial: todas las pantallas dependían de HU-20a y HU-20b, y la primera
    pantalla de negocio (HU-21) recién entró el 26/10, a mitad del Sprint. HU-20a necesitó un
    segundo PR.
  - Un defecto llegó a `main` (FIX-HU-21): la aplicación guardaba el proyecto con estado "Active"
    y el Backlog solo aceptaba "Activo". Las pruebas no lo detectaron porque creaban los
    proyectos directo en la base, donde el valor por defecto era el correcto. Además, el error
    500 no quedaba registrado en el log.
  - La integración se concentró al final: HU-15, HU-17 y HU-25 entraron en los últimos tres días.
  - Se empezaron historias del Sprint 3 antes de planificarlas: el Planning Poker del Sprint 3-4
    se hizo el 31/10, cuando ya había cinco de esas historias integradas, y sus Issues no tienen
    milestone. El tablero no mostraba qué entraba en cada Sprint y los Sprints 2 y 3 se
    superpusieron.
  - Las estimaciones del documento de planificación no coinciden con el Planning Poker (HU-20a:
    5 y 8; HU-21: 5 y 3; HU-14: 5 y 3; HU-15: 8 y 5).
  - Se subieron archivos generados por error (el perfil de cobertura en HU-25) y se editó una
    evidencia desde la web de GitHub (cobertura de HU-10).
- **Acciones para el próximo sprint:**
  1. Cada historia que dependa de datos de otra suma al menos una prueba que cree esos datos con
     el servicio real, no insertándolos directo en la base. Responsable: cada Product Builder; lo
     controla quien revisa el PR.
  2. Ninguna historia se empieza sin estimación de Planning Poker ni milestone del Sprint en su
     Issue. Asignar ya los milestones "Sprint 3" y "Sprint 4". Responsable: Franco.
  3. Abrir el PR apenas la historia está lista y no dejar integraciones para el último día del
     Sprint. Responsable: Franco (lo sigue en las dailies).
  4. Igualar las estimaciones del documento de planificación con las del Planning Poker.
     Responsable: Franco.
  5. Registrar las dailies en este archivo, una línea por sesión. Responsable: Franco.
  6. Agregar el `.gitattributes` y el comando `go test -count=1 -p 1 ./...` al README.
     Responsable: (asignar).
  7. Las evidencias se generan y se suben desde la rama, nunca se editan desde la web de GitHub.
     Responsable: todos.

---

## Sprint 3

**Sprint Goal:** Al cierre del Sprint 3, el equipo puede estimar una historia con el Planning Poker
completo (votar en secreto, revelar los votos, repetir rondas y registrar el acuerdo), registrar y
resolver defectos, modificar un proyecto y consultar su estado y sus Sprints anteriores, y ver la
calidad y el avance de cada Sprint finalizado, con el contexto de trabajo protegido.

**Fechas:** del 27 de octubre al 1 de noviembre de 2026. Se superpuso con el final del Sprint 2
(ver la retrospectiva del Sprint 2).

**Historias:** HU-05a (5), HU-05b (3), HU-06 (8), HU-08 (8), HU-10 (8), HU-11 (8), HU-12 (3),
HU-16 (3), HU-18 (5), HU-19 (3) y HU-26 (3): 57 SP según el Planning Poker del 31/10 (milestone
"Sprint 3"). Además, FIX-HU-21 (defecto del Sprint 2, sin Story Points). HU-08 cubre la
"visualización" que la cátedra fija para el Sprint 3.

**Fecha de cierre:** 1 de noviembre de 2026 (al integrar HU-08).

### Seguimiento

Las dailies del Sprint 3 tampoco se registraron por escrito. El avance se reconstruye desde las
fechas de integración a `main`:

| Fecha | Integrado a `main` |
|---|---|
| 28/10 | HU-26 (#42), HU-05a (#46), HU-06 (#48) |
| 29/10 | FIX-HU-21 (#58), HU-11 (#54), HU-05b (#55) |
| 30/10 | (sin integraciones del Sprint 3; entraron HU-15 y HU-17 del Sprint 2) |
| 31/10 | HU-12 (#60), HU-18 (#61), HU-19 (#62); Planning Poker del Sprint 3-4 |
| 01/11 | HU-16 (#65), HU-10 (#66), HU-08 (#68) |

---

### Sprint Review

- **Fecha:** 1 de noviembre de 2026
- **Historias completadas:**

  | Historia | SP | Responsable | PR |
  |---|---|---|---|
  | HU-05a Emisión de Voto Oculto en Planning Poker | 5 | Alejandro | #46 |
  | HU-05b Revelación de Votos y Detección de Divergencia | 3 | Adriel | #55 |
  | HU-06 Registro y Trazabilidad de Defectos | 8 | Adriel | #48 |
  | HU-08 Visualización de Dashboard del Proyecto | 8 | Giuliana | #68 |
  | HU-10 Cálculo de Métricas de Calidad y Avance | 8 | Ignacio | #66 |
  | HU-11 Modificación de Proyecto | 8 | Alejandro | #54 |
  | HU-12 Consulta del Estado de un Proyecto | 3 | Franco | #60 |
  | HU-16 Consulta de Sprints Anteriores | 3 | Alejandro | #65 |
  | HU-18 Planning Poker: Nueva Ronda y Estimación Acordada | 5 | Alejandro | #61 |
  | HU-19 Resolución de Defectos | 3 | Giuliana | #62 |
  | HU-26 Firmar las cookies de contexto de trabajo | 3 | Alejandro | #42 |
  | FIX-HU-21 Proyecto creado desde la web rechazado por el Backlog | – | Franco | #58 |

  - Total: **57 SP completados sobre 57 planificados** (11 de 11 historias, 100,00 %).
  - Velocidad: **57 SP** (Sprint 1: 24 SP; Sprint 2: 44 SP).
- **Historias no completadas (y por qué):** ninguna.
- **Calidad del incremento:**
  - El pipeline de CI quedó en verde en todos los merges del Sprint.
  - Cobertura por paquete al integrar HU-08 (`specs/008-dashboard/cobertura.txt`): backlog 89,5 %,
    defect 87,8 %, effort 92,5 %, metrics 85,9 %, poker 88,5 %, project 87,3 %, sprint 83,3 % y
    web 90,1 %. Todos los paquetes de reglas de negocio y cálculos superan el 80 % del DoD.
- **Resultado frente al Sprint Goal:** cumplido. Desde el navegador se puede estimar una historia
  con el Planning Poker completo (HU-05a, HU-05b y HU-18), registrar y resolver defectos (HU-06 y
  HU-19), modificar un proyecto y consultar su estado y sus Sprints anteriores (HU-11, HU-12 y
  HU-16) y ver la calidad y el avance de cada Sprint finalizado (HU-10), con las cookies de
  contexto firmadas (HU-26). Además, el Dashboard (HU-08) reúne en una pantalla, con gráficos, el
  estado del proyecto, la velocidad, la desviación y los defectos abiertos.
- **Requisitos de la cátedra:** con este incremento quedan cubiertos los puntos 1 a 8 del
  enunciado. Falta el 9 (Reportes en PDF), que es el objetivo del Sprint 4.
- **Trabajo adelantado:** HU-13 (Consulta y Edición del Product Backlog, 8 SP) se integró el 01/11
  (#67), antes de cerrar el Sprint 3. No se cuenta en esta velocidad; se registra en el Sprint 4.
- **Feedback del Product Architect:** (completar luego de la presentación)

---

### Retrospectiva

- **Fecha:** 1 de noviembre de 2026
- **Acciones de la retro anterior:**

  | # | Acción | Estado |
  |---|---|---|
  | 1 | Pruebas que crean los datos con el servicio real | (confirmar con cada responsable) |
  | 2 | Planning Poker y milestone antes de empezar una historia | Parcial: el milestone "Sprint 3" se asignó al cierre; HU-13 se integró sin milestone |
  | 3 | No dejar las integraciones para el último día | No cumplida: el 31/10 y el 01/11 entraron 6 de las 11 historias (30 de 57 SP) |
  | 4 | Igualar las estimaciones del documento con las del Planning Poker | (confirmar) |
  | 5 | Registrar las dailies | No cumplida |
  | 6 | `.gitattributes` y comando de pruebas en el README | No cumplida |
  | 7 | No editar evidencias desde la web de GitHub | No cumplida: commit "Update tasks.md" en HU-13 |

- **Qué funcionó bien:**
  - Se completó otra vez el 100 % de lo comprometido, y la velocidad siguió subiendo (24 → 44 →
    57 SP).
  - El CI quedó en verde en los 13 merges del Sprint y la cobertura de reglas de negocio está
    entre 83 % y 93 %.
  - El Planning Poker captó el crecimiento de las historias: HU-06, HU-10 y HU-09 pasaron de 5 a
    8 SP entre la estimación inicial (25/09) y la del 31/10, porque el refinamiento les sumó
    escenarios.
  - El "Orden de implementación" de cada spec permitió encadenar dependencias el mismo día: HU-10
    entró a las 13 h y HU-08, que depende de ella, a las 19 h.
  - Ningún PR lo mergeó su propio autor.
  - Ya no hubo conflictos grandes al integrar: cada rama trajo `main` antes de su PR (HU-08 integró
    HU-13 minutos antes del merge).
- **Qué no funcionó / mejorar:**
  - La revisión de los PR no queda registrada: solo 4 de los 13 PR del Sprint (#48, #54, #55 y #62)
    tienen una aprobación en GitHub. El resto se mergeó sin review, aunque el DoD pide "revisado y
    aprobado por al menos un integrante distinto del autor". Es evidencia que la cátedra puede
    pedir.
  - La integración volvió a concentrarse al final del Sprint (6 historias en los últimos dos días).
  - Los Sprints siguen superpuestos: HU-13, del Sprint 4, se empezó y se integró antes de cerrar el
    Sprint 3, y su Issue (#45) sigue abierto aunque el PR ya se mergeó.
  - La carga quedó despareja: un integrante tomó 5 de las 11 historias (24 de 57 SP), y otros
    tomaron una o dos. Si esa persona se bloquea, el Sprint entero queda en riesgo.
  - La documentación del proceso quedó atrasada: no hay dailies escritas, el acta del Sprint 2 no
    se subió a tiempo y `docs/informe-metricas.md` sigue con la cobertura del 27/10. Las tres cosas
    forman parte del DoD del Sprint y de la entrega final.
  - Se cumplieron muy pocas acciones de la retro anterior: se comprometieron más acciones de las
    que el equipo podía sostener.
- **Acciones para el Sprint 4** (Sprint de un día; se eligen pocas y de impacto directo en la
  entrega):
  1. Todo PR del Sprint 4 se aprueba con "Approve" en GitHub por alguien distinto del autor antes
     del merge. Responsable: todos; lo controla Franco.
  2. Cerrar el Issue #45 (HU-13) y asignar el milestone "Sprint 4" a HU-13, HU-09, MEJORA-01, HU-27
     y HU-28, con su Sprint Goal. Asignar hoy un responsable a HU-09 y a HU-28. Responsable: Franco.
  3. Subir este archivo y actualizar `docs/informe-metricas.md` con la cobertura por paquete.
     Responsable: Franco.
  4. Fijar una hora límite de merge a `main` antes de la presentación. Lo que no esté integrado a
     esa hora no entra (HU-27 y HU-28 primero). Responsable: Franco.
  5. Preparar la demostración de trazabilidad con HU-10 (Historia → spec → criterios → escenarios
     BDD → tests → código Go). Responsable: Ignacio, autor de HU-10 (a confirmar).
  6. Las acciones 5, 6 y 7 de la retro anterior que no se cumplieron se registran como deuda del
     proceso y no se arrastran al Sprint 4.

---

## Sprint 4

**Sprint Goal:** Al cierre del Sprint 4, el equipo presenta al Product Architect el producto
completo: se puede editar el Product Backlog, exportar en PDF el reporte de un Sprint y el del
proyecto, y recorrer todas las pantallas con un diseño consistente, con la documentación final, el
informe de métricas y cobertura y la demostración de trazabilidad listos para la defensa.

**Fechas:** del 1 al 2 de noviembre de 2026. La Sprint Review final es la presentación del
2/11 por la tarde.

**Historias:**

| Historia | SP | Responsable | Compromiso |
|---|---|---|---|
| HU-13 Consulta y Edición del Product Backlog | 8 | Alejandro | Comprometida (integrada el 01/11, #67) |
| HU-09 Exportación de Reportes en PDF | 8 | (asignar) | Comprometida |
| MEJORA-01 Diseño visual consistente | 3 | Ignacio | Comprometida |
| HU-27 Identificación con Contraseña | 5 | Adriel | Objetivo extra (Could have) |
| HU-28 Roles del Proyecto (mínimos) | 8 | (asignar) | Objetivo extra (Could have) |

- Comprometido: 19 SP (HU-13, HU-09 y MEJORA-01). Objetivo extra: 13 SP (HU-27 y HU-28).
- Orden de merge: HU-09 → MEJORA-01 (cambios de `layout.html`) → HU-27 → HU-28. HU-27 cambia la
  identificación que usan las pruebas de las demás historias, así que va última. HU-28 depende de
  ella.
- Si HU-27 o HU-28 no llegan a la hora límite, vuelven al Product Backlog con el motivo
  registrado, como pide el DoD del Sprint.

**Entregables del DoD final:** reporte PDF del proyecto (HU-09), informe de métricas y cobertura
(≥ 80 %), documentación técnica y manual de usuario completos, y demostración de trazabilidad.