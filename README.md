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

### INTEGRANTES:
- Franco Valentin Velazco - Scrum Master
- Adriel Alonso Gonzalez - Product Builder
- Alejandro Ruiz Andreola - Product Builder
- Giuliana Betitol Rivadeo - Product Builder
- Juan Ignacio Sanchez - Product Builder

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
  backlog (HU-02), `003` Sprints (HU-04), `004` métricas (HU-07), `005` esfuerzo (HU-03).
- No subas a ningún archivo del repositorio (README, specs, `docker-compose.yml`) tu clave
  personal. Va solo en tu `docker-compose.override.yml`, que está en el `.gitignore`.