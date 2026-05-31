package database

import (
	"log"
	"sort"

	"gorm.io/gorm"
)

type Migration struct {
	ID  string
	SQL string
}

var Migrations = []Migration{
	{
		ID: "202605310001_request_workflow_constraints_indexes",
		SQL: `
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE requests
  ALTER COLUMN status SET DEFAULT 'PENDING_OPS',
  ALTER COLUMN status SET NOT NULL;

UPDATE requests SET status = 'PENDING_OPS' WHERE status IS NULL OR status = '';

ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_status_check;
ALTER TABLE requests ADD CONSTRAINT requests_status_check
  CHECK (status IN ('PENDING_OPS', 'PENDING_MM', 'ASSIGNED', 'FOR_DOCKING', 'DOCKED', 'CONFIRMED', 'REJECTED', 'CANCELED'));

CREATE INDEX IF NOT EXISTS idx_requests_status_timestamp ON requests (status, request_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_requests_timestamp_desc ON requests (request_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_requests_updated_timestamp ON requests (updated_at DESC, request_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_requests_plate_trgm ON requests USING gin (plate_number gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_requests_trip_trgm ON requests USING gin (linehaul_trip_no gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_requests_driver_trgm ON requests USING gin (driver_id gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_requests_cluster_trgm ON requests USING gin (cluster gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_requests_region_trgm ON requests USING gin (region gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_requests_dock_trgm ON requests USING gin (dock_no gin_trgm_ops);
`,
	},
	{
		ID: "202605310002_user_passwords_and_role_constraints",
		SQL: `
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash varchar(255);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('fte_ops', 'fte_mm', 'ops_pic', 'dock_officer', 'doc_officer', 'data_team', 'admin'));
`,
	},
}

func RunMigrations(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (id text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`).Error; err != nil {
		return err
	}

	migrations := append([]Migration(nil), Migrations...)
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].ID < migrations[j].ID })
	for _, migration := range migrations {
		var applied int64
		if err := db.Table("schema_migrations").Where("id = ?", migration.ID).Count(&applied).Error; err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(migration.SQL).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO schema_migrations (id) VALUES (?)`, migration.ID).Error
		}); err != nil {
			return err
		}
		log.Println("Applied database migration", migration.ID)
	}
	return nil
}
