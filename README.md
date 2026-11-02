## Descripción del Proyecto

**Software Metrics & Estimation** es una aplicación para administrar proyectos de software, 
permitiendo su estimación, planificación, seguimiento y medición de calidad, aplicando 
prácticas ágiles y de ingeniería de software.

### Objetivo

Desarrollar una aplicación que permita gestionar proyectos de software de punta a punta: 
desde la creación del Product Backlog y la organización en Sprints, hasta el registro de 
esfuerzo, defectos y el cálculo de métricas de seguimiento, presentadas en un dashboard 
con reportes exportables.

### Funcionalidades principales

- **Gestión de proyectos**: alta, modificación y consulta de proyectos e integrantes.
- **Product Backlog**: historias de usuario con prioridad, estado, story points y criterios 
  de aceptación.
- **Gestión de Sprints**: creación, Sprint Goal, asignación de historias, cierre y consulta 
  de sprints anteriores.
- **Estimación**: Story Points y mecanismo de Planning Poker (estimaciones ocultas, rondas, 
  detección de diferencias).
- **Registro de esfuerzo**: horas trabajadas por integrante/actividad, comparadas contra lo estimado.
- **Gestión de defectos**: severidad, estado, historia relacionada, sprint de detección/resolución.
- **Métricas**: velocidad del equipo, desviación esfuerzo estimado vs. real, % de historias 
  completadas, defectos detectados/resueltos, entre otras.
- **Dashboard**: visualización gráfica del estado del proyecto.
- **Reportes**: exportación de informes de proyecto/sprint en PDF.

### Stack tecnológico

- **Lenguaje**: Go (Golang) — núcleo y reglas de negocio.
- **Metodología**: Scrum.
- **SDD** (Specification-Driven Development) para especificar funcionalidades antes de implementarlas.
- **BDD** (Behavior-Driven Development) con escenarios Given–When–Then.
- **TDD** (Test-Driven Development), ciclo RED → GREEN → REFACTOR.
- **Git** para control de versiones.
- Herramientas de **IA** como soporte al desarrollo (código, tests, documentación, revisión).

### Metodología de trabajo

El proyecto se organiza en Sprints (Sprint 0 a Sprint 4), gestionados mediante GitHub Projects, 
con roles Scrum adaptados a la cátedra:

- **Product Architect**: los profesores de la cátedra.
- **Agile Enabler** (Scrum Master): un integrante del equipo.
- **Product Builders**: los integrantes que construyen el producto.

Las actas de Sprint Review y retrospectiva están en [`docs/retros.md`](docs/retros.md), el
informe de métricas y cobertura en [`docs/informe-metricas.md`](docs/informe-metricas.md) y el
manual de usuario en [`docs/manual-usuario.md`](docs/manual-usuario.md).

### INTEGRANTES:
- Franco Valentin Velazco - Agile Enabler (Scrum Master)
- Adriel Alonso Gonzalez - Product Builder
- Alejandro Ruiz Andreola - Product Builder
- Giuliana Bettiol Rivadeo - Product Builder
- Juan Ignacio Sanchez - Product Builder

## Arquitectura y documentación técnica

### Vista general

Es una aplicación web en Go que se renderiza en el servidor (sin JavaScript) y guarda los datos en
PostgreSQL. Las reglas de negocio viven en paquetes de dominio que no saben nada de HTTP ni de la
base de datos; la web y PostgreSQL se conectan a ellos a través de interfaces (puertos y
adaptadores).

```mermaid
flowchart LR
    N[Navegador] -->|HTTP| W["src/web<br/>handlers, middleware,<br/>plantillas html/template"]
    W --> S["Servicios de dominio<br/>src/project, backlog, sprint,<br/>poker, effort, defect, metrics,<br/>report, auth, role"]
    S -->|interfaz Store| A["Adaptadores PostgreSQL<br/>postgres_*_store.go"]
    A --> DB[(PostgreSQL 16)]
    W -->|interfaz Catalog| C["Catálogos de lectura<br/>postgres_*_catalog.go"]
    C --> DB
    S -->|src/report usa| P["src/pdf<br/>generador de PDF propio"]
    CMD["src/cmd/set-password<br/>src/cmd/set-role"] --> S
```

### Paquetes

| Paquete | Responsabilidad | Historias |
|---|---|---|
| `src/project` | Proyectos, integrantes, fechas y estado del proyecto | HU-01, HU-11, HU-12 |
| `src/backlog` | Historias del Product Backlog, Story Points, horas estimadas y edición | HU-02, HU-13, HU-17 |
| `src/sprint` | Sprints, Sprint Goal, asignación de historias, estados del Tablero y cierre | HU-04, HU-14, HU-15 |
| `src/poker` | Planning Poker: votos ocultos, revelación, divergencia, rondas y consenso | HU-05a, HU-05b, HU-18 |
| `src/effort` | Registro de horas trabajadas por integrante, fecha y actividad | HU-03 |
| `src/defect` | Defectos: alta, severidad, estado, resolución y reapertura | HU-06, HU-19 |
| `src/metrics` | Velocidad, desviación de esfuerzo, calidad y avance | HU-07, HU-10 |
| `src/report` | Contenido de los reportes de Sprint y de proyecto | HU-09 |
| `src/pdf` | Generador de PDF propio (sin dependencias externas) | HU-09 |
| `src/auth` | Contraseñas (PBKDF2-SHA256) y bloqueo por intentos fallidos | HU-27 |
| `src/role` | Roles del proyecto: Scrum Master e Integrante | HU-28 |
| `src/web` | Servidor HTTP, pantallas, formularios, contexto de trabajo, permisos y errores | HU-20a, HU-20b, HU-21 a HU-26, HU-08, HU-16, MEJORA-01 |
| `src/cmd/server` | Arranque: configuración, conexión, verificación del esquema y registro de las rutas | HU-20a |
| `src/cmd/set-password`, `src/cmd/set-role` | Comandos de administración para contraseñas y roles | HU-27, HU-28 |
| `db` | Migraciones SQL (`db/migrations`, embebidas en el binario) | todas |
| `test` | Escenarios BDD de punta a punta (Given/When/Then contra el servidor HTTP) | todas |

### Decisiones de diseño

- **Puertos y adaptadores.** Cada paquete de dominio define su interfaz `Store` y un servicio que
  la usa (`NewService(store)`). La implementación real es `NewPostgreSQL…Store`; para las pruebas
  unitarias hay dobles en memoria (`test_store.go`), así las reglas se prueban con TDD sin base de
  datos. `src/cmd/server/main.go` es el único lugar donde se conectan las piezas.
- **Lectura separada de la escritura en la web.** Las pantallas leen con "catálogos"
  (`*_catalog.go`) y escriben a través de los servicios de dominio, que aplican las validaciones.
- **Manejo de errores.** El dominio devuelve errores con nombre (`ErrGoalRequired`,
  `ErrActiveSprintExists`, …). La web los traduce a un mensaje para el usuario y a un código HTTP:
  422 para datos inválidos, 409 para conflictos, 404 si no existe, 403 sin permiso, 503 si la base
  no responde y 500 en otro caso, siempre con una página de error sin detalles internos. Los
  detalles se registran en el log con `log/slog`, sin la clave de la base.
- **Interfaz sin JavaScript.** Plantillas `html/template` (escapan el HTML) y CSS propio en
  `src/web/static/app.css`, ambos embebidos con `go:embed`. Los gráficos del Dashboard son SVG
  generados en el servidor, con una tabla equivalente al lado.
- **Seguridad.** Protección contra envíos de otro origen en todos los POST (HU-20b), cookies de
  contexto firmadas con HMAC (HU-26, clave `COOKIE_SECRET`), contraseñas con sal y PBKDF2 y
  verificación en tiempo constante (HU-27) y control del rol Scrum Master en el servidor para las
  acciones de planificación (HU-28).
- **Base de datos.** Migraciones versionadas con `golang-migrate` (`up` y `down`). Al arrancar, el
  servidor compara la versión del esquema con la última migración: si falta aplicar alguna, no
  arranca y muestra el comando para aplicarlas. Los reportes PDF leen todo en una sola transacción `REPEATABLE READ` de solo lectura,
  para que los números sean coherentes entre sí.
- **Dependencias mínimas.** Go 1.27.1, la biblioteca estándar y un solo módulo externo: el driver
  `pgx` para PostgreSQL. El PDF se genera con código propio (`src/pdf`).

### Estructura del repositorio

```text
src/                código Go: dominio, web y comandos
db/migrations/      migraciones SQL numeradas (up y down)
test/               escenarios BDD automatizados
specs/NNN-nombre/   SDD de cada historia: spec.md, plan.md, tasks.md, quickstart.md y evidencias
docs/               actas, informe de métricas, manual de usuario y deuda técnica
.github/workflows/  CI: gofmt, go vet, migraciones, pruebas con -race y cobertura
```

### Pruebas

- **Unitarias** (`*_test.go` junto al código): reglas de negocio, cálculos y validaciones con
  dobles en memoria.
- **De integración** (`*_integration_test.go`): adaptadores contra PostgreSQL real. Se omiten si
  `DATABASE_URL` no está definida.
- **BDD** (`test/*_bdd_test.go`): cada escenario Given/When/Then de las specs, de punta a punta
  contra el servidor HTTP.

```powershell
docker compose up -d
go test ./... -count=1 -p 1
```

`-p 1` evita que los paquetes que usan la misma base corran en paralelo. Los números de cobertura
están en [`docs/informe-metricas.md`](docs/informe-metricas.md).

### Trazabilidad

Cada historia se puede recorrer de punta a punta. Ejemplo con HU-10 (Cálculo de Métricas de
Calidad y Avance):

| Paso | Dónde |
|---|---|
| Historia de usuario | Issue #12 y `specs/010-quality-metrics/spec.md` ("Historia de usuario") |
| Especificación SDD | `specs/010-quality-metrics/spec.md` y `plan.md` |
| Criterios de aceptación | `specs/010-quality-metrics/spec.md` ("Escenarios BDD" y "Criterios de éxito") |
| Escenarios BDD | `test/web_quality_bdd_test.go` |
| Pruebas unitarias (TDD) | `src/metrics/quality_test.go`, `src/metrics/quality_service_test.go`, `src/web/quality_*_test.go` |
| Evidencia RED y GREEN | `specs/010-quality-metrics/evidencia-red.txt` y `evidencia-pruebas.txt` |
| Código Go | `src/metrics/quality.go`, `src/metrics/quality_service.go`, `src/web/quality_handlers.go` |

Las specs más recientes (por ejemplo HU-09, HU-13, HU-27 y HU-28) tienen además una sección
"Trazabilidad" que lista cada escenario con su prueba.

## Guía para comenzar el Sprint

Todos deben seguir este orden desde la raíz del repositorio:

1. Actualizar el repositorio:

  ```powershell
  git pull
  ```

  Esto descarga el módulo Go, Docker Compose, las especificaciones y las migraciones que ya
  estén en `main`.

2. Instalar Go `1.27.1` para Windows de 64 bits desde el instalador oficial:
  [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi).
  Después abrir una nueva terminal de PowerShell y verificar:

  ```powershell
  Test-Path "C:\Program Files\Go\bin\go.exe"
  $env:Path += ";C:\Program Files\Go\bin"
  go version
  ```
  
2.5. Descargar las dependencias ya declaradas en `go.mod` (no hace falta volver a
  agregarlas, eso ya está hecho — solo bajarlas a tu máquina):

```powershell
  go mod download
```

3. Instalar `golang-migrate` desde PowerShell si no está instalado:

  ```powershell
  go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  $env:Path += ";$(go env GOPATH)\bin"
  migrate -version
  ```

  Si muestra `dev`, es válido para esta instalación desde `@latest`: indica que el binario
  funciona, aunque no esté mostrando un número de release. También puede verificarse con:

  ```powershell
  migrate -help
  ```

  La segunda línea agrega la carpeta de herramientas de Go al `PATH` de la terminal actual.
  Para dejarlo permanente en Windows, agregá `$(go env GOPATH)\bin` a la variable de entorno
  `Path` del usuario y abrí una nueva terminal.

4. Levantar PostgreSQL:

  ```powershell
  docker compose up -d
  $env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable"
  migrate -path db/migrations -database $env:DATABASE_URL up
  ```

  Si tu `docker-compose.override.yml` usa otro usuario, clave o puerto, ajustá la URL y
  verificá la conexión con la sección "Base de datos" de más abajo.

  **Clave de las cookies (HU-26)**: el proyecto y el integrante activos viajan en cookies
  firmadas con `COOKIE_SECRET`. Debe tener **al menos 32 caracteres**; con menos, la aplicación
  no arranca. Si no está definida, se usa una clave temporal y las sesiones se pierden al
  reiniciar (no es un error). Para conservarlas entre reinicios:

  ```powershell
  $env:COOKIE_SECRET = "una-clave-larga-de-al-menos-32-caracteres"
  ```

  No subas tu clave al repositorio.

  **Contraseñas de los integrantes (HU-27)**: para identificarse hace falta una contraseña.
  Después de migrar (la migración 015 agrega `members.password_hash`), el Scrum Master configura
  la de cada integrante con el comando, que pide la contraseña dos veces (de 8 a 128 caracteres):

  ```powershell
  go run ./src/cmd/set-password -integrante 1
  ```

  El número es el `id` del integrante en `members`. Sin contraseña configurada, ese integrante
  no puede identificarse. Volver a correr el comando reemplaza la contraseña.

  Limitaciones conocidas: el bloqueo por 5 intentos fallidos vive en memoria y se reinicia con el
  servidor; la contraseña se ve en la consola al escribirla en el comando; no hay cambio ni
  recuperación de contraseña desde la web.

  **Roles del proyecto (HU-28)**: cada integrante es "Scrum Master" o "Integrante" en cada
  proyecto. Solo el Scrum Master puede crear Sprints, agregar o quitar sus historias, cerrar el
  Sprint, modificar el proyecto y, en Planning Poker, abrir y cerrar rondas, abrir una nueva ronda
  y registrar la estimación acordada. Votar, mover historias en el Tablero y registrar esfuerzo
  siguen abiertos a cualquier integrante identificado. La migración 016 agrega
  `project_members.role` y deja a todos como "Integrante", así que **después de migrar hay que
  asignar al menos un Scrum Master por proyecto**:

  ```powershell
  go run ./src/cmd/set-role -proyecto 1 -integrante 1 -rol "Scrum Master"
  ```

  Los números son el `id` del proyecto en `projects` y del integrante en `members`. El rol se
  escribe exactamente "Scrum Master" o "Integrante" (`-rol "Integrante"` le saca el rol de Scrum
  Master). El comando responde, por ejemplo, `Beto ahora es Scrum Master de «Alfa»`, y el cambio
  vale desde la siguiente solicitud, sin volver a identificarse. Un proyecto puede tener más de un
  Scrum Master; si no tiene ninguno, sus acciones de planificación muestran "Este proyecto no
  tiene Scrum Master asignado". Los botones siguen visibles para todos: el control lo hace el
  servidor.

5. Aplicar las migraciones pendientes en orden numérico. Revisen el `quickstart.md` de su
  historia: allí figuran `DATABASE_URL`, el comando exacto y las migraciones previas requeridas.

  Los ejemplos de `DATABASE_URL` de los `quickstart.md` usan el puerto `5432` solo como
  referencia: usen el puerto de su contenedor (ver la sección "Base de datos" más abajo).

6. Abrir `specs/00X-nombre-de-su-historia/tasks.md` y ejecutar las tareas en orden. Las pruebas
  deben escribirse antes del código, siguiendo el ciclo TDD solicitado.

⚠️ **Importante:** antes de programar o correr tests, siempre asegurate de tener
PostgreSQL levantado:
```powershell
docker compose up -d
```
Podés chequear si ya está corriendo con `docker compose ps`. Docker Desktop **no**
arranca los contenedores solo al prender la compu — hay que levantarlo a mano cada
sesión de trabajo, salvo que lo configures para que inicie automático.

7. Crear una rama propia antes de modificar archivos:

  ```powershell
  git checkout -b HU-02
  ```

  Reemplacen `HU-02` por la historia asignada y realicen commits a medida que completen tareas.

Ante dudas sobre `tasks.md` o bloqueos técnicos, comuníquense con el equipo antes de avanzar
sobre una decisión de diseño.


## Base de datos: verificar la conexión y traer migraciones de otras ramas

> Todos los comandos son para PowerShell, ejecutados desde la raíz del repositorio.
> Reemplazá `USUARIO`, `CLAVE` y `PUERTO` por **tus** datos (los de tu `docker-compose.override.yml`,
> o `sym_user` / `sym_pass` / `5433` si no tenés override).

### A. Verificar que tu base responde con tus datos

Hacé estos tres chequeos en orden. Si uno falla, andá a la sección B.

**1. Levantá la base y mirá en qué puerto quedó publicada**

```powershell
docker compose up -d postgres
docker compose port postgres 5432
```

El resultado es algo como `0.0.0.0:5433`. El número después de los dos puntos es tu `PUERTO`.

**2. Verificá que el usuario existe dentro de la base**

```powershell
docker exec seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion -c "select current_user"
```

Tiene que devolver una tabla con tu usuario. Este chequeo **no verifica la clave**; solo que el usuario y la base existen.

**3. Verificá la clave y el puerto desde tu máquina**

```powershell
$env:DATABASE_URL = "postgres://USUARIO:CLAVE@localhost:PUERTO/seguimiento_y_medicion?sslmode=disable"
migrate -path db/migrations -database $env:DATABASE_URL version
```

Este es el chequeo que importa, porque conecta por el puerto publicado como lo hacen las pruebas de Go.

- `no migration`: conectó bien y la base está vacía, todavía sin migraciones. Está todo correcto.
- Un número (por ejemplo `1`): conectó bien y esa es la última migración aplicada.
- Cualquier mensaje de error: ver la tabla de la sección B.

> La variable `$env:DATABASE_URL` solo dura mientras esa ventana de PowerShell esté abierta.
> Para dejarla fija, usá `setx DATABASE_URL "..."` con la misma URL y reiniciá VS Code.

Para ver qué datos está leyendo Docker de la combinación de `docker-compose.yml` y
`docker-compose.override.yml`, corré `docker compose config`. **No compartas esa salida ni
capturas con ella: muestra tu clave.**

### B. Si algo falla

| Mensaje | Causa probable | Qué hacer |
|---|---|---|
| `role "USUARIO" does not exist` | La base se creó antes con otros datos | Ver B.1 |
| `password authentication failed` | La clave de la URL no es la de la base | Ver B.1 |
| `connection refused` o `timeout` | Puerto equivocado o Docker apagado | Revisá el paso A.1, abrí Docker Desktop y repetí `docker compose up -d postgres` |
| `database "seguimiento_y_medicion" does not exist` | La base se creó con otro nombre | Ver B.1 |
| Conecta, pero a otra base | Hay otro PostgreSQL ocupando el puerto 5432 | Usá el puerto de A.1 y no el 5432 |

#### B.1. El usuario o la clave del override no coinciden con la base

PostgreSQL crea el usuario, la clave y la base **solo la primera vez que arranca con el volumen
vacío**. Si cambiaste `POSTGRES_USER`, `POSTGRES_PASSWORD` o `POSTGRES_DB` en tu
`docker-compose.override.yml` después de haber levantado Docker una vez, el cambio **no se
aplica** a una base ya creada.

Tenés dos salidas:

**Opción 1: usar los datos con los que se creó la base originalmente.** Normalmente son los del
`docker-compose.yml`: `sym_user` / `sym_pass`. Probá el chequeo A.2 con esos datos. Si
funciona, usá esos en tu `DATABASE_URL`.

**Opción 2: recrear la base con tus datos.** Esto **borra todo lo que haya en tu base local**
(solo la tuya, no afecta a nadie más). Hacelo únicamente si no tenés datos que quieras conservar:

```powershell
docker compose down -v
docker compose up -d postgres
```

Después repetí los chequeos de la sección A y aplicá las migraciones de nuevo:

```powershell
migrate -path db/migrations -database $env:DATABASE_URL up
```

> Regla para todo el equipo: si cambiás usuario, clave o nombre de base en tu override
> **después** de haber levantado la base, hay que hacer `docker compose down -v`.

### C. Traer la migración de un compañero sin esperar su merge

Sirve cuando necesitás una tabla creada por otra historia (por ejemplo, proyectos o historias del
backlog) para probar la tuya y su rama todavía no está en `main`.

**1. Traé lo que hay en GitHub y mirá qué ramas existen**

```powershell
git fetch origin
git branch -r
```

**2. Mirá qué migraciones tiene la rama del compañero** (cambiá `HU-01` por su rama)

```powershell
git ls-tree -r --name-only origin/HU-01 db/migrations
```

**3. Parado en tu propia rama** (confirmá con `git branch`; la actual tiene un asterisco),
traé cada archivo que necesites, el `.up.sql` y el `.down.sql`:

```powershell
git checkout origin/HU-01 -- db/migrations/001_create_projects.up.sql
git checkout origin/HU-01 -- db/migrations/001_create_projects.down.sql
```

**4. Guardalos en tu rama**

```powershell
git add db/migrations
git commit -m "chore: incorpora migracion 001 de HU-01"
```

**5. Aplicalas**

```powershell
migrate -path db/migrations -database $env:DATABASE_URL up
```

**Reglas**

- Nadie edita una migración que no es suya. Si hace falta un cambio, se le pide al dueño o se
  agrega una migración nueva.
- Si el dueño actualiza su archivo, repetí los pasos 3 y 4: traen la versión nueva. Si ya la
  habías aplicado, revertila antes con `migrate -path db/migrations -database $env:DATABASE_URL down 1`
  y volvé a aplicarla.
- Si `migrate` queda en estado "dirty", avisen al equipo antes de usar `force`.
- Orden y dueños de las migraciones: `001` proyectos e integrantes (HU-01), `002` historias del
  backlog (HU-02), `004` Sprints (HU-04), `007` métricas (HU-07), `003` esfuerzo (HU-03).
- No subas a ningún archivo del repositorio (README, specs, `docker-compose.yml`) tu clave
  personal. Va solo en tu `docker-compose.override.yml`, que está en el `.gitignore`.

## Levantar el servidor web

Con la base levantada y las migraciones aplicadas:

```powershell
$env:ADDR = ":8081"   # opcional: por defecto usa :8080
go run ./src/cmd/server
```

Después abrí `http://localhost:8081` (o `http://localhost:8080` si no definiste `ADDR`) en el
navegador. El uso de cada pantalla está en el [manual de usuario](docs/manual-usuario.md).