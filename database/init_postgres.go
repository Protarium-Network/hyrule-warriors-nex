package database

import (
	"os"

	"github.com/Protarium-Network/hyrule-warriors-nex/globals"
)

// initPostgres creates the tables this server owns directly: the
// hyrulewarriors_* leaderboard tables and the DataStore object store. Every
// statement is idempotent, so it is safe to run on every start.
//
// The matchmaking.* / tracking.* schema is NOT created here -
// nex-protocols-common-go v2.4.0 self-creates it inside
// CommonProtocol.SetManager (see nex/secure.go). Hand-authoring it first only
// risks a column-shape mismatch that no-ops the library's own CREATE TABLE
// IF NOT EXISTS. See docs/matchmaking-schema.md for the reference layout.
func initPostgres() {
	mustExec := func(label, query string) {
		if _, err := Postgres.Exec(query); err != nil {
			globals.Logger.Criticalf("%s: %s", label, err.Error())
			os.Exit(1)
		}
	}

	// --- Ranking (leaderboards) --------------------------------------------
	mustExec("hyrulewarriors_rankings", `CREATE TABLE IF NOT EXISTS hyrulewarriors_rankings (
		owner_pid   bigint,
		unique_id   bigint,
		category    bigint,
		score       bigint,
		order_by    smallint,
		update_mode smallint,
		groups      bytea,
		param       bigint,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid, category)
	)`)

	mustExec("hyrulewarriors_ranking_categories", `CREATE TABLE IF NOT EXISTS hyrulewarriors_ranking_categories (
		category   bigint PRIMARY KEY,
		order_by   smallint NOT NULL CHECK (order_by IN (0, 1)),
		created_at bigint NOT NULL
	)`)

	mustExec("hyrulewarriors_common_datas", `CREATE TABLE IF NOT EXISTS hyrulewarriors_common_datas (
		unique_id   bigint,
		owner_pid   bigint,
		common_data bytea,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid)
	)`)

	mustExec("hyrulewarriors ranking indexes", `
		CREATE INDEX IF NOT EXISTS hyrulewarriors_rankings_category_score_idx
			ON hyrulewarriors_rankings (category, score, updated_at);
		CREATE INDEX IF NOT EXISTS hyrulewarriors_rankings_owner_category_idx
			ON hyrulewarriors_rankings (owner_pid, category);
		CREATE INDEX IF NOT EXISTS hyrulewarriors_common_datas_owner_updated_idx
			ON hyrulewarriors_common_datas (owner_pid, updated_at DESC)
	`)

	// --- DataStore (Network Link / My Fairy objects) --------------------
	mustExec("datastore schema", `CREATE SCHEMA IF NOT EXISTS datastore`)
	mustExec("datastore sequence", `CREATE SEQUENCE IF NOT EXISTS datastore.object_data_id_seq
		INCREMENT 1 MINVALUE 1 MAXVALUE 281474976710656 START 1 CACHE 1`)
	mustExec("datastore.objects", `CREATE TABLE IF NOT EXISTS datastore.objects (
		data_id                      bigint NOT NULL DEFAULT nextval('datastore.object_data_id_seq') PRIMARY KEY,
		upload_completed             boolean NOT NULL DEFAULT FALSE,
		deleted                      boolean NOT NULL DEFAULT FALSE,
		owner                        bigint,
		size                         int,
		name                         text,
		data_type                    int,
		meta_binary                  bytea,
		permission                   int,
		permission_recipients        int[],
		delete_permission            int,
		delete_permission_recipients int[],
		flag                         int,
		period                       int,
		refer_data_id                bigint,
		tags                         text[],
		persistence_slot_id          int,
		extra_data                   text[],
		access_password              bigint NOT NULL DEFAULT 0,
		update_password              bigint NOT NULL DEFAULT 0,
		creation_date                timestamp,
		update_date                  timestamp
	)`)

	globals.Logger.Success("Postgres schema ready (ranking + datastore; matchmaking.* is library-created)")
}
