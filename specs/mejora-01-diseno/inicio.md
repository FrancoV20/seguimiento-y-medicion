# MEJORA-01 · Escenario 7: Pantalla de Inicio

Ampliación de la MEJORA-01 (Issue #63), pedida por el equipo el 02/11/2026: la pantalla de Inicio debe
presentar la aplicación y ordenar las secciones, para usarla en la defensa. No cambia ningún comportamiento:
solo la presentación de `GET /`.

## Escenario BDD

**Escenario 7: Inicio con secciones bien diferenciadas**
- **Dado** la aplicación con sus secciones registradas
- **Cuando** abro el Inicio
- **Entonces** veo:
  - la bienvenida "Bienvenido a Software Metrics & Estimation", con la etiqueta "Scrum · SDD · BDD · TDD" y una
    descripción de la aplicación;
  - los accesos "Elegir proyecto" (o "Cambiar proyecto" si ya hay uno activo) y "Crear un proyecto";
  - "Cómo se trabaja", con los 5 pasos en orden: Proyecto, Backlog, Sprint, Trabajo diario, Cierre y métricas;
  - las secciones agrupadas en "Planificar", "Ejecutar" y "Medir", cada una con su ícono, su nombre (enlace a
    la sección) y una descripción corta;
  - el pie "UTN FRSR · Ingeniería y Calidad de Software · 2026".

## Reglas

- R1: solo se muestran las secciones registradas (las del menú). Un grupo sin secciones no se muestra.
- R2: una sección registrada que no pertenece a ningún grupo aparece en "Otras secciones"; ninguna se pierde.
- R3: "Inicio" no aparece entre las secciones (ya es la pantalla actual).
- R4: sin secciones registradas se mantiene el mensaje "Todavía no hay secciones disponibles.".
- R5: sin recursos externos ni JavaScript: íconos SVG en línea con `aria-hidden="true"` (el nombre de la sección
  es el texto accesible) y degradé hecho con CSS.
- R6: a 400 px de ancho no hay desplazamiento horizontal; las columnas se apilan.

## Grupos

| Grupo | Subtítulo | Secciones, en este orden |
|---|---|---|
| Planificar | Antes y al comenzar cada Sprint | Proyectos, Backlog, Estimaciones, Planning Poker, Sprints |
| Ejecutar | Durante el Sprint | Tablero, Registro de esfuerzo, Defectos, Cierre de Sprint |
| Medir | Al cerrar y en la Review | Dashboard, Estado del proyecto, Métricas del Sprint, Calidad y avance, Sprints anteriores, Reportes |

## Trazabilidad

| Regla | Prueba |
|---|---|
| Escenario 7 | `TestWebHomeBDD_InicioConSeccionesDiferenciadas` (test/web_home_bdd_test.go) |
| R1, R2, R3 | `TestHomeGroups_AgrupaSoloLasSeccionesRegistradas`, `TestHomeGroups_SeccionSinGrupoVaAOtras` |
| R4 | `TestHomeGroups_SinSecciones`, `TestHomePage_SinSeccionesMantieneElMensaje` |
| Bienvenida, pasos y accesos | `TestHomePage_BienvenidaPasosYAccesos`, `TestHomePage_ConProyectoActivoOfreceCambiarlo` |
| R5 | `TestHomeGroups_IconosSinRecursosExternos`, `TestPage_NoReferenciaOtrosOrigenes` (existente) |
| R6 | Verificación manual a 400 px (evidencia.txt) |
