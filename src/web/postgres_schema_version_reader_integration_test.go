package web

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// T013: PostgreSQLSchemaVersionReader contra PostgreSQL real.
// Trabaja en un esquema temporal propio para no tocar el schema_migrations de desarrollo.

func openSchemaTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	url := os.Getenv(EnvDatabaseURL)
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	admin, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	schema := fmt.Sprintf("hu20a_schema_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatalf("no se pudo crear el esquema temporal: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE") })

	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	db, err := sql.Open("pgx", url+sep+"search_path="+schema)
	if err != nil {
		t.Fatalf("no se pudo abrir la base con el esquema temporal: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, schema
}

func TestPostgreSQLSchemaVersionReader_SinTablaEsVersionCero(t *testing.T) {
	db, _ := openSchemaTestDB(t)

	status, err := NewPostgreSQLSchemaVersionReader(db).ReadSchemaStatus(context.Background())

	if err != nil || status != (SchemaStatus{}) {
		t.Errorf("sin schema_migrations se esperaba versión 0, se obtuvo %+v (error: %v)", status, err)
	}
}

func TestPostgreSQLSchemaVersionReader_LeeVersionYDirty(t *testing.T) {
	db, _ := openSchemaTestDB(t)
	// Misma estructura que crea golang-migrate para PostgreSQL.
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)`); err != nil {
		t.Fatalf("no se pudo crear schema_migrations: %v", err)
	}
	reader := NewPostgreSQLSchemaVersionReader(db)

	if _, err := db.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (7, false)`); err != nil {
		t.Fatalf("no se pudo insertar la versión: %v", err)
	}
	status, err := reader.ReadSchemaStatus(context.Background())
	if err != nil || status != (SchemaStatus{Version: 7}) {
		t.Errorf("se esperaba versión 7 limpia, se obtuvo %+v (error: %v)", status, err)
	}

	if _, err := db.Exec(`UPDATE schema_migrations SET dirty = true`); err != nil {
		t.Fatalf("no se pudo marcar dirty: %v", err)
	}
	status, err = reader.ReadSchemaStatus(context.Background())
	if err != nil || status != (SchemaStatus{Version: 7, Dirty: true}) {
		t.Errorf("se esperaba versión 7 dirty, se obtuvo %+v (error: %v)", status, err)
	}
}

func TestPostgreSQLSchemaVersionReader_TablaVaciaEsVersionCero(t *testing.T) {
	db, _ := openSchemaTestDB(t)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)`); err != nil {
		t.Fatalf("no se pudo crear schema_migrations: %v", err)
	}

	status, err := NewPostgreSQLSchemaVersionReader(db).ReadSchemaStatus(context.Background())

	if err != nil || status != (SchemaStatus{}) {
		t.Errorf("con la tabla vacía se esperaba versión 0, se obtuvo %+v (error: %v)", status, err)
	}
}
