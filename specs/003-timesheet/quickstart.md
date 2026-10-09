# Quickstart: Registrar horas trabajadas

## Prerequisites

- Go 1.27.1 instalado en Windows desde [go1.27.1.windows-amd64.msi](https://go.dev/dl/go1.27.1.windows-amd64.msi); verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Herramienta `golang-migrate/migrate` instalada; seguir la instalación de Windows documentada en [README.md](../../README.md).
- Paquete `src/effort`, `PostgreSQLEffortStore` y `go.mod` creados según el plan.
- Variable `DATABASE_URL` configurada para PostgreSQL, por ejemplo:
   `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
   Si el `README.md` usa el puerto `5433`, reemplazar `5432` por `5433`.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Aplicar las migraciones pendientes en orden numérico. La migración 005 depende de que las
migraciones 001, que crea los proyectos e integrantes, 002, que crea las historias del backlog,
y 003, que crea los Sprints, ya estén aplicadas porque los registros de esfuerzo referencian
historias e integrantes existentes y solo se aceptan en historias de un Sprint activo:

```powershell
migrate -path db/migrations -database "$DATABASE_URL" up
```

Las pruebas de integración de HU-07 (migración 004) leen la tabla `effort_logs`, por lo que la
migración 005 debe estar aplicada antes de ejecutarlas.

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias de esfuerzo, los
escenarios BDD y la integración de `PostgreSQLEffortStore` contra PostgreSQL.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Carga de esfuerzo diario:
   - una historia asignada a un Sprint activo recibe 4 horas con fecha actual y actividad
     "Desarrollo de endpoint en Go";
   - el esfuerzo real acumulado aumenta exactamente 4 horas;
   - varios registros sobre la misma historia se acumulan sin sobrescribirse.
2. Horas inválidas:
   - horas negativas, iguales a cero y mayores a 24 se rechazan;
   - 24 y 0,5 se aceptan; 24,01 y valores con más de dos decimales se rechazan;
   - el mensaje es "El valor de horas no es válido" y el acumulado no cambia.
3. Actividad vacía:
   - actividad vacía o solo espacios se rechaza;
   - el mensaje indica que la descripción de la actividad es obligatoria.
4. Fecha faltante:
   - fecha omitida se rechaza con el mensaje de fecha requerida;
   - fecha posterior a la actual se rechaza.
5. Integrante no especificado:
   - integrante omitido, inexistente o ajeno al proyecto se rechaza;
   - el mensaje indica que se debe especificar el integrante responsable.
6. Historia sin Sprint activo:
   - una historia sin Sprint o con un Sprint no activo rechaza la carga.
7. Varios errores a la vez:
   - se informan todos los errores de validación juntos y no se guarda nada.
8. Error al guardar:
   - ante una falla de almacenamiento se informa "La operación no pudo completarse";
   - no hay mensaje de éxito y el acumulado no cambia.
9. PostgreSQL:
   - `PostgreSQLEffortStore` usa pgx mediante `database/sql`;
   - guarda el registro y devuelve el acumulado dentro de una transacción;
   - la restricción `CHECK` rechaza horas fuera de rango escritas directamente;
   - una falla transaccional no deja registros parciales;
   - la prueba de integración se ejecuta contra PostgreSQL real.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar las reglas reales de validación, aritmética y persistencia, no
reemplazarlas con mocks o métodos exclusivos de test.
