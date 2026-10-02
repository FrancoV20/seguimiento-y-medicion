# Quickstart: Crear nuevo proyecto

## Prerequisites

- Go 1.22 o posterior instalado.
- Docker Desktop instalado y en ejecución.
- PostgreSQL levantado con Docker Compose para las pruebas de integración.
- Repositorio ubicado en su raíz.
- Paquete `src/project`, adaptador `PostgreSQLProjectStore` y `go.mod` creados según el plan.
- Variables de conexión PostgreSQL configuradas según el adaptador de pruebas.

## Run the tests

Desde la raíz del repositorio, iniciar PostgreSQL:

```powershell
docker compose up -d postgres
```

Luego ejecutar las pruebas:

```powershell
go test ./...
```

El comando debe finalizar correctamente y ejecutar las pruebas unitarias del dominio y los
escenarios de aceptación disponibles.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Creación exitosa:
   - nombre no vacío;
   - al menos un integrante disponible;
   - fecha de inicio y fin informadas;
   - fin posterior o igual al inicio;
   - `ProjectStore.Create` recibe el proyecto;
   - el resultado confirma éxito y estado `Activo`.
2. Fechas inválidas:
   - fecha de fin anterior rechaza la creación;
   - no se llama a `ProjectStore.Create`;
   - se muestra un error de inconsistencia temporal.
3. Fechas iguales:
   - fecha de inicio igual a fecha de fin permite la creación.
4. Campos obligatorios:
   - nombre vacío se rechaza;
   - integrantes ausentes se rechazan;
   - fecha de inicio o fin ausente se rechaza;
   - los datos restantes se conservan para corregir el formulario.
5. Persistencia:
   - un error de `ProjectStore` no muestra éxito;
   - no se confirma el proyecto como creado;
   - un reintento válido no genera duplicación dentro del flujo del caso de uso.
6. Integrantes:
   - un integrante inexistente se rechaza;
   - integrantes duplicados se rechazan con un error de validación explícito;
   - los integrantes duplicados no se normalizan ni se ignoran en silencio;
   - al menos un integrante válido queda asociado al proyecto.
7. PostgreSQL:
   - `PostgreSQLProjectStore` usa pgx mediante `database/sql`;
   - el proyecto y sus integrantes se persisten en PostgreSQL;
   - un fallo transaccional no deja registros parciales.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio.
Las pruebas deben ejecutar la validación real del proyecto y del período, sin reemplazarla con
mocks o métodos exclusivos de test.
