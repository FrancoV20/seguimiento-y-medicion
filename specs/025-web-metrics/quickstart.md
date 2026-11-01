# Quickstart: Interfaz web de métricas del Sprint

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-07, HU-15, HU-17, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada (la leen la aplicación y las pruebas de integración; sin ella, las pruebas
  contra PostgreSQL se omiten con `t.Skip`). Con el `docker-compose.yml` del repositorio:

  ```powershell
  $env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
  ```

- **Nota del puerto**: `docker-compose.yml` publica PostgreSQL en el puerto **5433** de tu máquina (`5433:5432`),
  para no chocar con un PostgreSQL instalado en Windows que use el 5432. El pipeline de GitHub Actions usa el
  **5432** porque levanta su propio servicio. Si tu `docker-compose.override.yml` cambia usuario, clave o puerto,
  usá esos datos; el puerto real lo muestra `docker compose port postgres 5432`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`. No compartir la salida de
  `docker compose config`: muestra la clave.

## Run the application

Esta feature no agrega migraciones; basta con las ya aplicadas:

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Levantar la aplicación (la variable vale para esa ventana de PowerShell):

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

**Nota de puertos**: la aplicación escucha en el puerto **8080** por omisión (`ADDR`); si está ocupado, usar otro con
`$env:ADDR = ":8081"` y reemplazar `8080` por ese número en las direcciones de abajo. PostgreSQL queda en el **5433**
de `docker-compose.yml` (o el de tu `docker-compose.override.yml`); no confundir los dos puertos.

### Probar a mano con un Sprint cerrado (T027)

Preparación: un proyecto con al menos un Sprint cerrado y, para el aviso, otro Sprint en curso.

1. Crear el proyecto en `http://localhost:8080/proyectos` (HU-21) y elegirlo en
   `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear tres historias en `/backlog` (HU-22) con Story Points 5, 3 y 13, y cargar horas estimadas en `/estimaciones`
   (HU-17): 4 y 6 horas en las dos primeras; la tercera, sin horas estimadas.
3. Armar el Sprint 1 en `/sprints` (HU-23) con las tres historias.
4. En `/tablero` (HU-14), pasar las dos primeras a "Hecho" y dejar la tercera "En progreso".
5. Identificarse como integrante y cargar horas en `/esfuerzo` (HU-24): 5 y 7 horas en las dos historias "Hecho".
6. Cerrar el Sprint 1 en `/cierre` (HU-15). La historia "En progreso" vuelve al Backlog.
7. Abrir `http://localhost:8080/metricas` y verificar:
   - el Sprint 1 en la lista, con su Sprint Goal y sus fechas (dd/mm/aaaa);
   - velocidad `8 Story Points` y desviación `+20,00 %` (10 horas estimadas y 12 reales), con la fórmula;
   - sin alerta de cálculo parcial (las dos historias "Hecho" tienen horas estimadas y reales).
8. Sin horas estimadas: repetir 2 a 6 con un Sprint 2 cuya única historia "Hecho" no tenga horas estimadas; abrir
   `/metricas`: el Sprint 2 es el elegido, la desviación dice "No disponible", la velocidad se ve y aparece la alerta
   "Cálculo parcial: basado en 0 de 1 historias".
9. Elegir otro Sprint: hacer clic en "Sprint 1" de la lista (`/metricas?sprint=ID`) y ver sus métricas.
10. Sprint en curso: crear un Sprint 3 sin cerrarlo; en `/metricas` no figura en la lista y arriba se lee
    "El Sprint 3 sigue en curso; sus métricas se calculan al cerrarlo".
11. Casos vacíos y de borde:
    - con un proyecto sin Sprints cerrados se ve "Aún no hay Sprints finalizados";
    - `/metricas?sprint=abc` o el identificador del Sprint en curso muestran el finalizado más reciente, sin error;
    - sin proyecto elegido (ventana privada), `/metricas` redirige a elegir el proyecto;
    - un Sprint Goal con `<script>` se ve como texto.

### Resultado de la verificación manual

Recorrido hecho el 01/11/2026 sobre el commit `8e564d2`, con Chromium real y los datos cargados desde las pantallas
de la aplicación. El detalle de cada paso (códigos HTTP, textos observados y conteos de la base) está en
[evidencia-manual.txt](evidencia-manual.txt).

| Paso | Resultado observado | Estado |
|---|---|---|
| 7 Sprint 1 cerrado | "Sprint 1 · Primer incremento · Del 01/09/2026 al 14/09/2026", "8 Story Points", "+20,00 %" con la fórmula, sin alerta | OK |
| 8 Sin horas estimadas | Sprint 2 elegido por defecto: "No disponible", "2 Story Points", "Cálculo parcial: basado en 0 de 1 historias" | OK |
| 9 Elegir otro Sprint | `/metricas?sprint=1`: "8 Story Points" y "+20,00 %"; "Sprint 1" marcado en la lista | OK |
| Velocidad cero (escenario 4) | Sprint 3 cerrado sin historias Hecho: "0 Story Points" y "No disponible" | OK |
| 10 Sprint en curso | "El Sprint 4 sigue en curso; sus métricas se calculan al cerrarlo"; la lista muestra solo los Sprints 3, 2 y 1 | OK |
| 11 Sin Sprints finalizados | "Aún no hay Sprints finalizados"; con un Sprint activo, además el aviso "El Sprint 1 sigue en curso…" | OK |
| 11 `sprint` inválido, activo o ajeno | `abc`, el activo, el de otro proyecto, `999`, `-1` y `01`: 200 con el Sprint 3 (el más reciente) | OK |
| 11 Sin proyecto elegido | 303 a `/contexto/proyecto?volver_a=%2Fmetricas`; al elegir, vuelve a `/metricas` | OK |
| 11 Sprint Goal con `<script>` | Se ve como texto, escapado en el HTML, sin diálogo | OK |
| `POST /metricas` | 405 "Acción no permitida"; conteos de la base iguales antes y después (sin filas en `sprint_metrics`) | OK |
| Sin JavaScript / sin internet | Métricas visibles con JavaScript desactivado; ningún recurso externo | OK |
| Base detenida | 503 "El servicio no está disponible temporalmente", sin métricas ni texto técnico; al volver la base, 200 sin reiniciar | OK |

Si alguna de esas pantallas no está disponible, cargar los mismos datos por SQL respetando el esquema de
`data-model.md` y cerrar el Sprint desde `/cierre`.

Las pruebas automáticas no dependen de las demás pantallas: crean sus propios proyectos, historias, Sprints, horas
y cierres.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Formato (unitarias): desviación positiva, negativa y cero con dos decimales, coma y signo; "No disponible";
   velocidad cero; alerta de cálculo parcial con X de Y; elección del Sprint (válido, inválido, activo, ajeno, sin
   lista).
2. Adaptador de HU-07 (servicio real y PostgreSQL real): velocidad de las historias "Hecho"; desviación con valores
   conocidos (por ejemplo 10 horas estimadas y 12 reales dan `+20,00 %`); sin horas estimadas; cálculo parcial con
   X de Y; Sprint cerrado por HU-15 sin historias completadas (velocidad cero); las historias devueltas no cuentan.
3. Catálogo (PostgreSQL real): solo Sprints finalizados del proyecto, más reciente primero; el Sprint activo no
   figura y su número se informa; otro proyecto no se mezcla; proyecto sin Sprints.
4. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - Sprint finalizado: velocidad y desviación correctas con datos reales; por defecto el más reciente;
   - sin horas estimadas: "No disponible" y velocidad visible;
   - cálculo parcial: alerta con X de Y;
   - velocidad cero: `0 Story Points` y "No disponible";
   - lista de Sprints y métricas del Sprint elegido con `?sprint=ID`; `sprint` inválido, activo o de otro proyecto se
     ignora;
   - sin Sprints finalizados, con y sin Sprint activo;
   - Sprint Goal con `<script>`: se ve como texto;
   - falla de lectura o de cálculo: página de error de la aplicación;
   - sin proyecto activo: redirección de HU-20b;
   - `POST` a `/metricas`: página 405.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas ejecutan las
reglas reales y el almacenamiento real; solo se usa una fuente que falla para simular las fallas de lectura y de
cálculo.

Guardar la evidencia en `specs/025-web-metrics/`. En PowerShell, ejecutar antes `chcp 65001` y
`[Console]::OutputEncoding = [System.Text.Encoding]::UTF8` para que las tildes y la ñ no salgan como `├®`
(en algunas terminales `chcp` solo no alcanza):

```powershell
chcp 65001
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la historia, la
ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/025-web-metrics/cobertura.txt
```

## Revisión de seguridad (T028)

Revisado `src/web/metrics_handlers.go`, `src/web/metrics_view.go`, `src/web/metrics_source.go` y
`src/web/postgres_metrics_catalog.go`:

- Toda falla de lectura o de cálculo pasa por `Renderer.Fail` de HU-20a: el navegador recibe solo `TitleInternal` y
  `MsgInternal` (500) o `TitleUnavailable` y `MsgUnavailable` (503). El detalle técnico va únicamente al registro del
  servidor; la prueba T031/T032 verifica que una consulta, una clave y una ruta del error no lleguen a la página.
- Los errores que arman el catálogo y el adaptador nombran la operación y los identificadores, nunca la
  `DATABASE_URL` ni la clave (la conexión ya viene abierta desde `main`). El mensaje del controlador de PostgreSQL que
  envuelven solo llega al registro del servidor, igual que en el resto de las pantallas de HU-20a.
- `RegisterMetrics` devuelve un texto fijo y `main` informa "no se pudo registrar la sección Métricas del Sprint",
  sin detalles.
- `metrics_view.go` no registra ni muestra errores; sus textos son fijos o vienen de HU-07. Los datos ingresados
  (Sprint Goal) se escriben con `html/template`, que los escapa (T034).
- El parámetro `sprint` se valida con `parseID` de HU-20b y solo se usa para buscar en la lista del proyecto
  activo; nunca llega a una consulta. El proyecto sale siempre del contexto del servidor.
- La pantalla no usa JavaScript ni recursos externos y solo acepta `GET` (T035).