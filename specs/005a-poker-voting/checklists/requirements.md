# Checklist de calidad de la especificación: HU-05a

**Propósito**: validar que la especificación está completa antes de implementar.
**Creada**: 2026-10-27
**Feature**: [spec.md](../spec.md)

## Calidad del contenido

- [x] La historia sigue el formato "Como…, quiero…, para…".
- [x] Los escenarios no contienen detalles de implementación.
- [x] Enfocada en el valor para el Product Builder.
- [x] Todas las secciones obligatorias están completas (Sección SDD: intención, alcance, fuera de alcance,
  entradas, salidas, reglas, restricciones, condiciones de error).

## Completitud de requisitos

- [x] No quedan marcadores [NEEDS CLARIFICATION], "(a definir)" ni preguntas abiertas (3 dudas respondidas el
  27/10, ver "Clarifications").
- [x] Los 13 escenarios están en Dado/Cuando/Entonces simples, sin "Esquema del escenario" (el 8 se escribe como
  8a y 8b, uno por acción).
- [x] Cada escenario tiene al menos un requisito funcional que lo cubre.
- [x] Cada requisito es verificable y usa MUST/MUST NOT (FR-001 a FR-023).
- [x] Los criterios de éxito son medibles (SC-001 a SC-008).
- [x] Los casos límite están identificados (identificadores inválidos, otro proyecto, concurrencia, cierre
  simultáneo, escape, recarga, cambio de identificación).
- [x] Alcance y fuera de alcance definidos (revelar, divergencia, nueva ronda, estimación acordada, roles).
- [x] Dependencias explícitas: HU-01, HU-02, HU-20a, HU-20b; habilita HU-05b y HU-18.
- [x] Los códigos 303/422/409/404/500 están definidos por caso.
- [x] Los mensajes están definidos en un solo lugar (`src/web/poker_messages.go`).

## Preparación

- [x] La tabla de trazabilidad conecta escenario, requisito, éxito y prueba, con los mismos nombres de prueba que
  `tasks.md`.
- [x] Todos los requisitos y criterios de éxito tienen tareas en `tasks.md` (tabla "requisitos → tareas").
- [x] El plan respeta la constitución v1.1.0: puerto `poker.Store` y catálogo `web.PokerCatalog`, ambos con
  adaptador PostgreSQL y pruebas de integración; RED → GREEN por bloque con `evidencia-red.txt`.
- [x] La migración nueva (`011_create_poker`) tiene `up` y `down` y no edita migraciones ajenas.
- [x] El único archivo ajeno editado es `src/cmd/server/main.go`; `src/backlog` solo se usa en una prueba.
- [x] El ocultamiento de votos se garantiza por construcción (el modelo de vista no tiene campo de valor ajeno) y
  por prueba (página idéntica byte a byte al cambiar el voto ajeno).
- [ ] Aceptada por el equipo (completar al subir el Issue).

## Notas

- Decisiones del equipo tomadas el 27/10: cualquier historia del proyecto es elegible; el selector lista todas;
  cada ronda muestra a todos los integrantes con "Votó"/"Sin votar".
- Riesgo de numeración de migraciones documentado (`research.md`, decisión 1): T002 y T080 lo controlan.
- Estimación: 5 SP (la de la tarjeta). Con 83 tareas, una migración, un paquete nuevo y un catálogo, conviene
  revisarla en el Planning Poker.
