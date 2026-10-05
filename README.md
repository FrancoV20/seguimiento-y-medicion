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

  Esto descarga el módulo Go, Docker Compose, las migraciones y las especificaciones.

2. Instalar Go `1.27.1` para Windows de 64 bits desde el instalador oficial:
  [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi).
  Después abrir una nueva terminal de PowerShell y verificar:

  ```powershell
  Test-Path "C:\Program Files\Go\bin\go.exe"
  $env:Path += ";C:\Program Files\Go\bin"
  go version
  ```

3. Instalar `golang-migrate` desde PowerShell si no está instalado:

  ```powershell
  go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  $env:Path += ";$(go env GOPATH)\bin"
  migrate -version
  ```

  La segunda línea agrega la carpeta de herramientas de Go al `PATH` de la terminal actual.
  Para dejarlo permanente en Windows, agregá `$(go env GOPATH)\bin` a la variable de entorno
  `Path` del usuario y abrí una nueva terminal.

4. Levantar PostgreSQL:

  ```powershell
  docker compose up -d
  ```

5. Aplicar las migraciones pendientes en orden numérico. Revisen el `quickstart.md` de su
  historia: allí figuran `DATABASE_URL`, el comando exacto y las migraciones previas requeridas.

6. Abrir `specs/00X-nombre-de-su-historia/tasks.md` y ejecutar las tareas en orden. Las pruebas
  deben escribirse antes del código, siguiendo el ciclo TDD solicitado.

7. Crear una rama propia antes de modificar archivos:

  ```powershell
  git checkout -b feature/HU-02
  ```

  Reemplacen `HU-02` por la historia asignada y realicen commits a medida que completen tareas.

Ante dudas sobre `tasks.md` o bloqueos técnicos, comuníquense con el equipo antes de avanzar
sobre una decisión de diseño.
