package database

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"ariga.io/atlas/sql/migrate"
	atlasmysql "ariga.io/atlas/sql/mysql"

	_ "github.com/go-sql-driver/mysql"
)

func MigrateHashGen() {
	dir, err := migrate.NewLocalDir("migrations")
	if err != nil {
		log.Fatal(err)
	}

	files, err := dir.Files()
	if err != nil {
		log.Fatal(err)
	}

	sum, err := migrate.NewHashFile(files)
	if err != nil {
		log.Fatal(err)
	}

	if err := migrate.WriteSumFile(dir, sum); err != nil {
		log.Fatal(err)
	}

	log.Println("atlas.sum regenerated, overall sum:", sum.Sum())
}

func RunMigration(dsn string, migrationFiles embed.FS) error {
	tmpDir, err := os.MkdirTemp("", "atlas-migrations-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	log.Printf("[migrate] tmpDir = %s", tmpDir)

	extracted := 0
	// extract embedded files
	if err := fs.WalkDir(migrationFiles, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			log.Printf("[migrate] mkdir %s", path)
			return os.MkdirAll(filepath.Join(tmpDir, path), 0755)
		}

		data, err := migrationFiles.ReadFile(path)
		if err != nil {
			return err
		}

		dst := filepath.Join(tmpDir, path)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		log.Printf("[migrate] extract %s -> %s (%d bytes)", path, dst, len(data))
		extracted++

		return os.WriteFile(dst, data, 0644)
	}); err != nil {
		return err
	}
	log.Printf("[migrate] total %d files extracted from embed.FS", extracted)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		return err
	}
	log.Printf("[migrate] db connection ok")

	driver, err := atlasmysql.Open(db)
	if err != nil {
		return err
	}

	migDir := filepath.Join(tmpDir, "migrations")
	dir, err := migrate.NewLocalDir(migDir)
	if err != nil {
		return err
	}

	debugChecksum(migDir, dir)

	rrw, err := newSQLRevisions(db)
	if err != nil {
		return err
	}
	log.Printf("[migrate] revisions table ready, ident = %q", rrw.Ident().Name)

	if existing, err := rrw.ReadRevisions(context.Background()); err != nil {
		log.Printf("[migrate] ReadRevisions error: %v", err)
	} else {
		log.Printf("[migrate] %d revision(s) already recorded:", len(existing))
		for _, rev := range existing {
			log.Printf("[migrate]   - version=%s type=%v applied=%d/%d", rev.Version, rev.Type, rev.Applied, rev.Total)
		}
	}

	ex, err := migrate.NewExecutor(
		driver,
		dir,
		rrw,
		migrate.WithAllowDirty(true),
		migrate.WithLogger(stepLogger{}),
	)
	if err != nil {
		return err
	}

	if pending, err := ex.Pending(context.Background()); err != nil {
		log.Printf("[migrate] Pending() error: %v", err)
	} else {
		log.Printf("[migrate] %d pending file(s):", len(pending))
		for _, f := range pending {
			log.Printf("[migrate]   - %s", f.Name())
		}
	}

	err = ex.ExecuteN(context.Background(), 0)
	log.Printf("[migrate] ExecuteN result: %v", err)
	return err
}

// debugChecksum compares the checksum freshly computed from the migration
// files on disk against what's recorded in atlas.sum, file by file, so we
// can pinpoint exactly which entry is causing the mismatch instead of just
// getting a generic "checksum mismatch" error.
func debugChecksum(dirPath string, dir *migrate.LocalDir) {
	files, err := dir.Files()
	if err != nil {
		log.Printf("[migrate-debug] dir.Files() error: %v", err)
		return
	}
	log.Printf("[migrate-debug] %d .sql file(s) found in %s:", len(files), dirPath)
	for _, f := range files {
		log.Printf("[migrate-debug]   - %s (version=%s, %d bytes)", f.Name(), f.Version(), len(f.Bytes()))
	}

	fresh, err := migrate.NewHashFile(files)
	if err != nil {
		log.Printf("[migrate-debug] NewHashFile error: %v", err)
		return
	}
	log.Printf("[migrate-debug] freshly computed overall sum: %s", fresh.Sum())

	raw, err := os.ReadFile(filepath.Join(dirPath, migrate.HashFileName))
	if err != nil {
		log.Printf("[migrate-debug] atlas.sum NOT FOUND in extracted dir: %v", err)
		return
	}
	log.Printf("[migrate-debug] atlas.sum raw content:\n%s", string(raw))

	var stored migrate.HashFile
	if err := stored.UnmarshalText(raw); err != nil {
		log.Printf("[migrate-debug] failed parsing atlas.sum: %v", err)
		return
	}
	log.Printf("[migrate-debug] atlas.sum stored overall sum: %s", stored.Sum())

	for _, entry := range fresh {
		want, err := stored.SumByName(entry.N)
		switch {
		case err != nil:
			log.Printf("[migrate-debug] %s -> MISSING from atlas.sum", entry.N)
		case want != entry.H:
			log.Printf("[migrate-debug] %s -> MISMATCH (atlas.sum=%s, actual=%s)", entry.N, want, entry.H)
		default:
			log.Printf("[migrate-debug] %s -> ok", entry.N)
		}
	}
}

// stepLogger prints every statement atlas executes, so we can see exactly
// how far it got (or whether it ran anything at all / took the baseline shortcut).
type stepLogger struct{}

func (stepLogger) Log(e migrate.LogEntry) {
	switch v := e.(type) {
	case migrate.LogExecution:
		log.Printf("[migrate] executing %d file(s) from %s to %s", len(v.Files), v.From, v.To)
	case migrate.LogFile:
		log.Printf("[migrate] -> file %s (skip=%d)", v.File.Name(), v.Skip)
	case migrate.LogStmt:
		log.Printf("[migrate]    stmt: %s", v.SQL)
	case migrate.LogError:
		log.Printf("[migrate]    ERROR: %v (sql=%q)", v.Error, v.SQL)
	case migrate.LogDone:
		log.Printf("[migrate] done")
	}
}

// sqlRevisions implements migrate.RevisionReadWriter backed by a plain
// MySQL table. It replaces migrate.NewEntRevisions, which only exists in
// atlas's internal CLI package (ariga.io/atlas/cmd/atlas/internal/migrate)
// and isn't importable as a library.
type sqlRevisions struct {
	db *sql.DB
}

const revisionsTable = "atlas_schema_revisions"

func newSQLRevisions(db *sql.DB) (*sqlRevisions, error) {
	const ddl = `CREATE TABLE IF NOT EXISTS ` + revisionsTable + ` (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		description VARCHAR(255) NOT NULL,
		type BIGINT UNSIGNED NOT NULL DEFAULT 2,
		applied BIGINT NOT NULL DEFAULT 0,
		total BIGINT NOT NULL DEFAULT 0,
		executed_at DATETIME NOT NULL,
		execution_time BIGINT NOT NULL DEFAULT 0,
		error TEXT NULL,
		error_stmt TEXT NULL,
		hash VARCHAR(255) NOT NULL DEFAULT '',
		partial_hashes JSON NULL,
		operator_version VARCHAR(255) NOT NULL DEFAULT ''
	)`
	if _, err := db.Exec(ddl); err != nil {
		return nil, err
	}
	return &sqlRevisions{db: db}, nil
}

func (r *sqlRevisions) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: revisionsTable}
}

const selectCols = `version, description, type, applied, total,
	executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version`

func (r *sqlRevisions) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+selectCols+` FROM `+revisionsTable+` ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []*migrate.Revision
	for rows.Next() {
		rev, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	return revs, rows.Err()
}

func (r *sqlRevisions) ReadRevision(ctx context.Context, v string) (*migrate.Revision, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+selectCols+` FROM `+revisionsTable+` WHERE version = ?`, v)
	rev, err := scanRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, migrate.ErrRevisionNotExist
	}
	return rev, err
}

func (r *sqlRevisions) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	hashes, err := json.Marshal(rev.PartialHashes)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO `+revisionsTable+`
			(version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			description = ?, type = ?, applied = ?, total = ?, executed_at = ?, execution_time = ?,
			error = ?, error_stmt = ?, hash = ?, partial_hashes = ?, operator_version = ?`,
		rev.Version, rev.Description, uint(rev.Type), rev.Applied, rev.Total, rev.ExecutedAt,
		rev.ExecutionTime.Milliseconds(), nullable(rev.Error), nullable(rev.ErrorStmt), rev.Hash, hashes, rev.OperatorVersion,
		rev.Description, uint(rev.Type), rev.Applied, rev.Total, rev.ExecutedAt,
		rev.ExecutionTime.Milliseconds(), nullable(rev.Error), nullable(rev.ErrorStmt), rev.Hash, hashes, rev.OperatorVersion,
	)
	return err
}

func (r *sqlRevisions) DeleteRevision(ctx context.Context, v string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM `+revisionsTable+` WHERE version = ?`, v)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRevision(s rowScanner) (*migrate.Revision, error) {
	var (
		rev             migrate.Revision
		typ             uint
		execMS          int64
		errStr, errStmt sql.NullString
		hashes          sql.NullString
	)
	if err := s.Scan(&rev.Version, &rev.Description, &typ, &rev.Applied, &rev.Total,
		&rev.ExecutedAt, &execMS, &errStr, &errStmt, &rev.Hash, &hashes, &rev.OperatorVersion); err != nil {
		return nil, err
	}
	rev.Type = migrate.RevisionType(typ)
	rev.ExecutionTime = time.Duration(execMS) * time.Millisecond
	rev.Error = errStr.String
	rev.ErrorStmt = errStmt.String
	if hashes.Valid && hashes.String != "" {
		_ = json.Unmarshal([]byte(hashes.String), &rev.PartialHashes)
	}
	return &rev, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
