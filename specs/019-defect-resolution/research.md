# Investigación: HU-19 – Resolver y reabrir defectos

## Decisiones

1. **Sin Sprint activo → bloquear.**
   - Decisión: 409 con mensaje claro, sin cambios.
   - Motivo: voto del equipo (22/10) y coherencia con HU-06, que tampoco registra
     defectos sin Sprint activo.
   - Alternativa descartada: permitir resolver sin Sprint (dejaría "Resuelto" sin
     Sprint de resolución y rompería el CHECK de HU-06).

2. **El Sprint de resolución es el Sprint activo.**
   - Motivo: el enunciado dice "durante un Sprint activo, registra ese Sprint".
     Elegirlo agregaría un campo sin valor claro.
   - Consecuencia: el escenario 4 es una regla defensiva; se mantiene por estar en
     el enunciado de la cátedra.

3. **Reabrir sin requerir Sprint activo.**
   - Motivo: reabrir no registra ningún Sprint.
   - Consecuencia: reabrir un defecto de un Sprint finalizado cambia el conteo de
     resueltos de ese Sprint en HU-10.

4. **Estados ya alcanzados → 409.**
   - Motivo: resolver dos veces pisaría el Sprint de resolución original y
     falsearía las métricas.

5. **404 para inexistente, inválido u otro proyecto.**
   - Motivo: mismo comportamiento para los tres casos; no revela defectos de otros
     proyectos.

6. **Un solo UPDATE de estado y Sprint de resolución.**
   - Motivo: respetar el CHECK de coherencia de HU-06.

7. **Bloqueos.** `FOR UPDATE` sobre el defecto evita doble resolución. `FOR SHARE`
   sobre el Sprint activo se serializa con el cierre de Sprint de HU-15 (que
   bloquea el Sprint con `FOR UPDATE`). El orden defecto → Sprint no forma ciclos
   con HU-06 ni con HU-15.

8. **Sin catálogo nuevo.** HU-06 ya entrega estado y Sprint de resolución en
   `DefectRow`; duplicarlo agregaría una consulta y un puerto sin valor.

9. **Formularios con campo `defecto`.** Mantiene todos los errores como
   re-render de la misma pantalla y no depende de rutas con parámetros.

10. **Sin token en el formulario.** HU-20b protege con un middleware global de
    origen (`CrossOriginGuard`); no existe un token que incluir.

11. **Reutilizar `ErrNoActiveSprint` de HU-06.** Mismo significado; el mensaje lo
    decide cada acción.

12. **Mensajes.** Texto propuesto por el equipo, en un único archivo; el de falla de
    guardado reutiliza `MsgOperationFailed` de HU-20b.

## Datos a verificar en T001 (tras el merge de HU-06)

- Nombres reales de: el tipo y el constructor del manejador de `/defectos`, la
  función que arma la vista, la función que registra la sección, los valores de
  `Status`, el puerto `Store`, los fixtures de las pruebas de integración y la
  ruta del archivo de textos.
- Que no existan ya `ErrDefectNotFound`, `ErrResolutionBeforeDetection`,
  `ErrAlreadyResolved`, `ErrAlreadyOpen`, `Resolved`, `ResolutionStore` ni
  `ResolutionService` (`git grep -n`).
- Que la migración de HU-06 sea la que figura en `data-model.md` (columnas
  `status`, `resolution_sprint_id`, `detection_sprint_id` y el CHECK).
- Cómo cierra HU-15 el Sprint (bloqueo de la fila) si ya está mergeada.

## Preguntas abiertas

Ninguna.