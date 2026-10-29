# Checklist de calidad de la especificación: HU-12

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
- [x] Los 10 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo son los del Issue y cierran aritméticamente (cantidades, totales, porcentajes y estados por
  fecha).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-015).
- [x] Los criterios de éxito son medibles (SC-001 a SC-005).
- [x] Los casos límite están identificados (redondeos, 100 % y 0 %, hora del día, zona horaria, Sprint fuera de las
  fechas del proyecto, historia devuelta al Backlog, estado guardado, recarga, escape, muchas historias).
- [x] Alcance y fuera de alcance definidos (días restantes, lista de Proyectos, métricas por Sprint, gráficos,
  edición).
- [x] Dependencias explícitas: HU-01, HU-14, HU-20a, HU-20b; HU-23 para la verificación manual.
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
- `/speckit-clarify` (28/10): sin preguntas; las decisiones del equipo y el Issue cubren todas las categorías.
- `/speckit-analyze` (28/10) no encontró problemas críticos ni violaciones de la constitución; cobertura de FR y SC
  100%. Se corrigieron:
  - C1, en T016 y la trazabilidad: `TestHU12_TextosEscapados` (nombre y objetivo escapados);
  - U1, en T009 y research.md §10: fechas del Sprint 1 de «Gamma».
  - C2 (Sprint activo fuera de las fechas del proyecto) y C3 (`HEAD`) quedan documentados sin prueba propia.
- Revisión de cuentas: los datos del Issue cierran (8 historias y 50,00 %; 3 de 3 y 100,00 %; 0 de 0 y 0,00 %; los
  cinco estados por fecha, incluidos los dos bordes); los casos de redondeo dan los valores de la tabla de
  data-model.md con la fórmula de enteros. Cada escenario usa su propio proyecto.
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
