# Checklist de calidad de la especificación: HU-09

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-28 · **Actualizada**: 2026-10-28 (reporte del proyecto)
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Scrum Master.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas.
- [x] Los 18 escenarios del Issue están en Dado/Cuando/Entonces simples, sin "Esquema del escenario".
- [x] Los datos de ejemplo cierran aritméticamente (historias, porcentajes, Story Points, horas, horas por
  integrante, velocidad, acumulada, promedio, desviación y defectos, del Sprint y del proyecto) y ningún Sprint se usa
  para dos repartos distintos.
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-030).
- [x] Los criterios de éxito son medibles (SC-001 a SC-007).
- [x] Los casos límite están identificados (`id` inválido, `confirmar` con otro valor, Sprint sin registro de cierre,
  resueltos detectados antes, defecto reabierto, desviación negativa y cero, decimales, sin esfuerzo, textos largos,
  caracteres fuera de WinAnsi, paréntesis, varias páginas, historia replanificada, `HEAD`; del proyecto: solo un
  Sprint en curso, historias sin Sprints, defectos en singular y todos resueltos, estado según fechas, sin
  integrantes, proyecto borrado).
- [x] Alcance y fuera de alcance definidos (gráficos, correo, guardar PDF, varios proyectos, opciones de formato).
- [x] Dependencias explícitas: HU-06, HU-07, HU-08, HU-10, HU-12, HU-15, HU-17, HU-20a, HU-20b; orden de
  implementación después de HU-08, HU-10 y HU-12.
- [x] Los códigos 200/303/404/405/500/503 están definidos por caso.
- [x] Los textos de la pantalla y de los PDF se definen en un solo lugar.

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas"), con 50
  tareas como máximo.
- [x] El plan respeta la constitución v1.1.0: puertos con adaptador PostgreSQL y pruebas de integración; RED → GREEN
  por bloque con `evidencia-red.txt`; sin dependencias externas para el PDF.
- [x] Sin migración nueva y sin editar migraciones existentes.
- [x] Los nombres de HU-06, HU-08, HU-10, HU-12, HU-15 y HU-17 que todavía no están en `main` quedan marcados para
  confirmar en T001, que frena si falta HU-08, HU-10 o HU-12.
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Las decisiones del equipo están registradas en "Clarifications" (sesión 2026-10-28), incluida la actualización del
  reporte del proyecto (DoD de la entrega final: Should Have, valor 70, 8 SP).
- `/speckit-clarify` (28/10): una pregunta. En un Sprint en curso, el avance y los Story Points planificados y
  completados también dicen "Se calculan al cerrar el Sprint" (R9, FR-009, FR-010).
- Primer `/speckit-analyze` (28/10): sin críticos; se corrigieron C1–C4, I1, I2 y U1 (Sprint sin registro de cierre,
  datos de T026, `HEAD`, Sprint sin esfuerzo, integrante fuera del proyecto, ancho en Bold, escenario 7).
- Segundo `/speckit-analyze` (28/10, con el reporte del proyecto): sin críticos ni violaciones de la constitución;
  cobertura de FR y SC 100%. Se corrigieron:
  - I1, en T012, T038 y T043: el almacén de prueba recibe `LoadProjectReport` en T038;
  - I2, en Supuestos: los proyectos de prueba «Delta», «Gamma» y «Largo»;
  - C1, en casos límite y en data-model: 404 si el proyecto se borra mientras se arma el reporte.
  - L1 (FR agrupados por reporte, sin orden numérico) y L2 (`[E#]` en lugar de `[US#]`) se mantienen.
- Revisión de cuentas: «Alfa» (Sprints 1, 2 y 3 y el proyecto: 1, 1, 1 y 4 historias, 57,14 %, 13 y 6,50, 38 h con
  2 sin estimar, 30 h con Ana 23 y Beto 7, 3 defectos con 2 abiertos), «Beta», «Gamma», «Delta» y «Largo» cierran.
  «Beta» del Issue no tiene Sprints: el Sprint ajeno del escenario 12 pasa a «Delta», y el integrante que dejó el
  proyecto, a «Caro» en «Delta» (Beto sigue en «Alfa»).
- Solo queda sin marcar "Aceptada por el equipo", que se completa al subir el Issue.
