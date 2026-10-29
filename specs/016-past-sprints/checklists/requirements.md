# Checklist de calidad de la especificación: HU-16

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 9 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo son los del Issue (y de HU-09) y cierran aritméticamente (orden de la lista, historias por
  tabla, Story Points y velocidades).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-015).
- [x] Los criterios de éxito son medibles (SC-001 a SC-005).
- [x] Los casos límite están identificados (identificadores inválidos, Sprint sin registro, "Hecho" sin Story Points,
  historia devuelta que entró a otro Sprint, título editado, singular, escape, otro proyecto, efecto de HU-07, envío de
  otro origen).
- [x] Alcance y fuera de alcance definidos (desviación y calidad, gráficos, exportar, modificar, reabrir).
- [x] Dependencias explícitas: HU-07, HU-15, HU-20a, HU-20b; HU-23 y HU-14 para la verificación manual.
- [x] Los códigos 200/303/404/405/500/503 están definidos por caso.
- [x] Los textos de la pantalla se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 30
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: puerto con adaptador PostgreSQL y pruebas de integración; RED →
  GREEN por bloque con `evidencia-red.txt`.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28).
- `/speckit-clarify` (28/10): sin preguntas; las decisiones del equipo, el Issue y las notas cubren todas las
  categorías. Lo que quedaba abierto se resolvió con esas decisiones y se anotó en "Casos límite": la historia "Hecho"
  sin Story Points cargada a mano da 500 porque falla HU-07; el título es el actual; la historia devuelta conserva
  "volvió al Backlog" aunque después haya entrado a otro Sprint.
- `/speckit-analyze` (28/10) no encontró problemas críticos ni violaciones de la constitución; cobertura de FR y SC
  100% (15 FR, 5 SC, 30 tareas). Se corrigieron:
  - U1, en T018: el 404 de un `{id}` inválido se prueba sin consultar la base (catálogo que cuenta llamadas);
  - U2, en T019: datos completos de `TestHU16_TextosEscapados` (historia "Hecho" asignada con 3 Story Points, para que
    HU-07 no falle);
  - L1, en la trazabilidad de spec.md: el Sprint sin registro (nota 5) tiene su propia fila, separada del escenario 5.
  - C1 (`HEAD`) y C2 ("Velocidad: 1 Story Points") quedan documentados sin prueba propia.
- Revisión de cuentas: los datos del Issue cierran:
  - lista [2, 1] sin el 3;
  - Sprint 2: 5 + 3 + 2 = 10 completados y «Reportes» 8 devuelta;
  - Sprint 1: 3 = 3, sin no completadas;
  - «Gamma» 1: velocidad 0 con dos devueltas;
  - «Gamma» 2: sin registro y velocidad 2 de HU-07.

  En todos los Sprints con registro, la velocidad de HU-07 (las "Hecho" asignadas) es igual a la suma de las
  completadas del registro.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
