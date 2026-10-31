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

## Verificación de T001 (31/10, con HU-06 y HU-15 en `main`)

### `src/defect` (HU-06)

| Supuesto | Nombre real |
|---|---|
| Valores de `Status` | `StatusOpen = "Abierto"`, `StatusResolved = "Resuelto"` (`model.go`) |
| Error sin Sprint activo | `ErrNoActiveSprint` (`errors.go`); se reutiliza |
| Puerto de registro | `Store` con `Create(ctx, RegisterInput) (Registered, error)` (`store.go`); no se toca |
| Servicio y adaptador | `Service` / `NewService(store)`; `PostgreSQLStore` / `NewPostgreSQLStore(db)` |
| Resultado de registrar | `Registered{ID int64, SprintNumber int64}` |
| Fixtures de integración | `openDefectTestDB(t)` y `newDefectStoreFixture(t, db)` en `postgres_store_integration_test.go` |

- `Registered.SprintNumber` es `int64`: `Resolved.SprintNumber` también será `int64` (en
  `data-model.md` figuraba `int`).
- `newDefectStoreFixture` crea el Sprint 1 activo y el Sprint 2 finalizado; el escenario 4 necesita
  el Sprint 3 finalizado (detección) y el Sprint 2 activo, así que HU-19 arma su propio fixture
  reutilizando `openDefectTestDB` y la misma limpieza (historias, Sprints y proyecto, en ese orden).

### `src/web` (HU-06 y HU-20)

| Supuesto | Nombre real |
|---|---|
| Manejador de `/defectos` | `defectsHandlers{server, service *defect.Service, catalog DefectCatalog}` (`defects_handlers.go`) |
| Registro de la sección | `RegisterDefects(s, service, catalog)`: `GET /defectos` y `POST /defectos` |
| Función que arma la vista | `pageData(r, projectID)`; se muestra con `renderPage(w, r, status, data, form, flash)` |
| Fila de la lista | `DefectRow{…, Status defect.Status, DetectionSprint int64, ResolutionSprint *int64}` (número de Sprint, `nil` = "—") |
| Archivo de textos de HU-06 | `defects_form.go` (`MsgDefect…`); `MsgOperationFailed` está en `forms.go` |
| Lectura de identificadores | `parseID(s) int64` en `context.go` (0 si no es válido) |
| Fixtures de pruebas web | `newDefectsHandlerFixture`, `selectProjectA` (`defects_handlers_test.go`); BDD: `givenDefectBDDScenario` (`test/web_defects_bdd_test.go`) |

Hallazgos que cambian el plan:

1. **`RegisterDefects` se usa en `main.go` y en 5 pruebas de HU-06.** Para no editar esas pruebas,
   se mantiene su firma y se agrega `RegisterDefectsWithResolution(s, service, resolution, catalog)`:
   `RegisterDefects` lo llama con `resolution = nil` y las rutas `POST /defectos/resolver` y
   `POST /defectos/reabrir` solo se registran cuando hay servicio de resolución. `main.go` usa la
   función nueva (T034 y T036).
2. **`renderFormStatus` de HU-06 siempre muestra `MsgDefectNoSprint` en un 409.** HU-19 no lo
   reutiliza: arma la pantalla con `pageData` + `renderPage` y su propio `Flash`.
3. **`GET /defectos` sin Sprint activo responde 409** (HU-06), con la lista y sin el formulario de
   registro. Después de reabrir sin Sprint activo, la redirección 303 lleva a esa pantalla: las
   pruebas de T038 lo esperan así.

### Verificaciones

- `git grep -n` de `ErrDefectNotFound`, `ErrResolutionBeforeDetection`, `ErrAlreadyResolved`,
  `ErrAlreadyOpen`, `Resolved`, `ResolutionStore` y `ResolutionService`: no existen. El único
  parecido es `web.ResolvedContext` (otro paquete y otro significado, sin choque).
- Migración `012_create_defects` coincide con `data-model.md`: `status` ('Abierto' | 'Resuelto'),
  `detection_sprint_id`, `resolution_sprint_id` y el CHECK de coherencia. No hace falta migración.
- HU-15 (en `main`) bloquea el Sprint con `SELECT … FROM sprints WHERE id = $1 FOR UPDATE` al
  cerrar: el `FOR SHARE` del Sprint activo en HU-19 se serializa con el cierre, como se planeó.

## Preguntas abiertas

Ninguna.