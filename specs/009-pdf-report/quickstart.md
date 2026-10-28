# Guía rápida: HU-09 – Exportación de Reportes en PDF

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-08, HU-10 y HU-12 en `main` (y con ellas HU-06 y HU-15), además de HU-07, HU-17, HU-20a y HU-20b. Comprobarlo
  con `git log --oneline main` y
  `git grep -n "func SummarizeVelocity\|func (s \*QualityService) SprintQuality\|CREATE TABLE defects\|CREATE TABLE sprint_story_results"`
  más la búsqueda de las reglas de estado y avance de HU-12 en `src/project` (T001).
- Rama: `git switch main; git pull; git switch -c HU-09`. Solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-09 no agrega migraciones.
- Para la verificación manual: un navegador (por ejemplo, Edge) y un lector de escritorio (por ejemplo, Adobe
  Acrobat Reader).

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "PDF|Report|HU09" -v ./src/pdf/... ./src/report/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/pdf/... ./src/report/...
gofmt -l src/pdf src/report src/web src/cmd/server test
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo.

## Escenarios de validación requeridos

Los valores esperados y sus cuentas están en "Datos de ejemplo" de [spec.md](spec.md) y en
[data-model.md](data-model.md).

1. **Generador** (`src/pdf`, sin base):
   - empieza con `%PDF-1.4` y termina con `%%EOF`;
   - cada entrada de la `xref` apunta al byte donde empieza `N 0 obj`, y `startxref` apunta a `xref`;
   - `/Count` y la cantidad de objetos `/Type /Page` coinciden con `PageCount()`;
   - "Página" aparece como los bytes `50 E1 67 69 6E 61`; "Login (v2)" como `Login \(v2\)`;
   - 120 filas pasan a varias páginas, el encabezado de la tabla está en todas y "Página X de Y" también;
   - `textWidth(Regular, 10, "Página")` = 31,13; una palabra más ancha que la columna se corta;
   - `ExtractText` devuelve el texto de una celda cortada en varias líneas como una sola frase.
2. **Reporte de un Sprint** (`src/report`, sin base): servicio con dobles (finalizado, en curso, cada falla,
   incoherencia, identificadores en 0); textos y formatos ("+6,25 %", "-8,50 %", "0,00 %", "75,00 %", "17", "2,5",
   singulares y plurales); `Render` con los textos en el orden de la spec.
3. **Reporte del proyecto** (`src/report`, sin base): servicio con dobles (con y sin Sprint en curso, sin Sprints,
   cada falla); textos ("3 defectos registrados: 2 abiertos y 1 resuelto", "6,50", "2 detectados y 1 resuelto");
   `RenderProject` con las seis partes en orden.
4. **Lectura** (PostgreSQL): el Sprint 2 de «Alfa» con 4 historias, 28 horas estimadas, 24 reales, Ana 17 h y Beto
   7 h y 2 defectos; el Sprint 3 en curso; el proyecto «Alfa» con 1, 1, 1 y 4 historias, 38 horas estimadas (2 sin
   estimar), 30 registradas (Ana 23 h y Beto 7 h), 3 defectos (2 abiertos); «Beta» vacío; nada de «Delta»;
   coherencia con HU-07, HU-08, HU-10 y HU-12.
5. **Pantalla**: escenarios 1 a 18 con los proyectos de prueba; `id` inválido; `confirmar` con otro valor; sin
   proyecto (303); `POST` (405); rendimiento con «Largo» (menos de 1 s por reporte).

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`), crear el proyecto «Alfa» (18/08/2026 al 30/11/2026) con los
   integrantes Ana y Beto, y elegirlo (`/contexto/proyecto`).
2. Con HU-22, HU-23, HU-17, HU-24, HU-14 y HU-15, armar «Alfa» como en "Datos de ejemplo" de la spec:
   - Sprint 1 con «Alta» (3 Story Points, 4 horas estimadas, Ana 4 h) terminada; cerrarlo;
   - Sprint 2 con «Login» (5, 10 h; Ana 8 y Beto 3), «Tablero» (3, 6 h; Ana 6), «Perfil» (2, sin horas; Ana 3 y
     Beto 1) terminadas, y «Reportes» (8, 12 h; Beto 3) "En progreso"; con HU-06 y HU-19, registrar los dos defectos
     del Sprint 2 y resolver el primero; cerrar el Sprint 2;
   - Sprint 3 activo con «Exportar» (5, 6 h; Ana 2) "En progreso" y «Ayuda» (sin estimar) "En Sprint", y el defecto
     "El PDF sale en blanco".

   Si HU-04 no acepta las fechas de la spec, usar otras: las cuentas no dependen de las fechas (salvo el estado del
   proyecto, que calcula HU-12 con la fecha del día).
3. Abrir `http://localhost:8080/reportes`: el botón "Exportar reporte del proyecto" y "Sprint 3" (En curso), "Sprint
   2" y "Sprint 1" (Finalizado), en ese orden, cada uno con "Exportar PDF".
4. Exportar el Sprint 2: se descarga `reporte-sprint-2.pdf`. Abrirlo en el navegador y en el lector de escritorio:
   - se abre sin avisos de archivo dañado y con los acentos bien ("Métricas", "Crítica", "contraseña");
   - encabezado con «Alfa», "Sprint 2", objetivo, "01/09/2026 al 14/09/2026", "Finalizado" y la fecha y hora de
     generación;
   - las 4 historias, «Reportes» con "En progreso – volvió al Backlog", «Perfil» con "Sin estimar" en horas, y "3 de
     4 historias completadas (75,00 %)";
   - "18 Story Points planificados y 10 completados", "28 horas estimadas en total (1 historia sin horas
     estimadas)";
   - "24 horas registradas en total", "Ana: 17 h", "Beto: 7 h";
   - "Velocidad: 10 Story Points", "Desviación de esfuerzo: +6,25 %", "Cálculo parcial: basado en 2 de 3 historias";
   - los 2 defectos y "2 defectos detectados y 1 resuelto en este Sprint";
   - "Página X de Y" en todas las páginas.
5. Exportar el Sprint 1: "No se registraron defectos en este Sprint".
6. Exportar el Sprint 3: aparece "El Sprint 3 sigue en curso: el reporte puede estar incompleto". "Cancelar" vuelve a
   "Reportes" sin descargar nada; "Generar igual" descarga `reporte-sprint-3.pdf` con "Reporte parcial: Sprint en
   curso", las historias en su estado actual y "Se calculan al cerrar el Sprint".
7. Presionar "Exportar reporte del proyecto": aparece "El proyecto tiene un Sprint en curso (Sprint 3): el reporte
   puede estar incompleto". "Cancelar" vuelve a "Reportes"; "Generar igual" descarga `reporte-proyecto-N.pdf`. Abrirlo
   en los dos lectores:
   - "Reporte parcial: hay un Sprint en curso", «Alfa», "18/08/2026 al 30/11/2026", el estado de HU-12, "Integrantes:
     Ana, Beto" y la fecha de generación;
   - historias por estado 1, 1, 1 y 4 y "4 de 7 historias completadas (57,14 %)";
   - las filas de los Sprints 1 ("1 de 1", "3 de 3", 3, "0,00 %", "0 detectados y 0 resueltos"), 2 ("3 de 4", "10 de
     18", 10, "+6,25 % (cálculo parcial)", "2 detectados y 1 resuelto") y 3 ("En curso: se calcula al cerrar el
     Sprint"), "Velocidad acumulada: 13 Story Points" y "Promedio: 6,50 Story Points por Sprint";
   - "38 horas estimadas en total (2 historias sin horas estimadas)", "30 horas registradas en total", "Ana: 23 h",
     "Beto: 7 h";
   - "3 defectos registrados: 2 abiertos y 1 resuelto" y los 2 abiertos;
   - "Página X de Y" en todas las páginas.
8. Crear el proyecto «Beta» sin historias ni Sprints, elegirlo y abrir "Reportes": "Aún no hay Sprints para
   exportar" y el botón. Exportar el reporte del proyecto: se descarga sin advertencia, con "Aún no hay Sprints", "0
   de 0 historias completadas (0,00 %)", "0 horas registradas en total" y "No se registraron defectos en este
   proyecto".
9. Abrir `/reportes/sprints/abc/pdf`, `/reportes/sprints/0/pdf` y la dirección de un Sprint de otro proyecto: página
   404, sin descarga.
10. Detener la base (`docker compose stop`) y exportar un Sprint y el proyecto: página de error 503, sin descarga.
11. Guardar el PDF del Sprint 2 como `specs/009-pdf-report/ejemplo-reporte-sprint.pdf` y el del proyecto «Alfa» como
    `specs/009-pdf-report/ejemplo-reporte-proyecto.pdf` para el PR.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/009-pdf-report/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/009-pdf-report/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/009-pdf-report/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Generador PDF, Dominio del reporte, Almacenamiento, Pantalla y
descarga, Reporte del proyecto), y cada sección se guarda antes de escribir el código de su bloque.
