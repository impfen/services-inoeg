// Kiebitz - Privacy-Friendly Appointment Scheduling
// Copyright (C) 2021-2021 The Kiebitz Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version. Additional terms
// as defined in section 7 of the license (e.g. regarding attribution)
// are specified at https://kiebitz.eu/en/docs/open-source/additional-terms.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package databases

import (
	"context"
	//"crypto/sha256"
	pg "github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4"
	"github.com/kiebitz-oss/services"
	"github.com/kiprotect/go-helpers/forms"
)

type PostgreSQL struct {
	ctx  context.Context
	pool *pg.Pool
}

type PostgreSQLSettings struct {
	ConnString string `json:"connection_string"`
}

var PostgreSQLForm = forms.Form{
	ErrorMsg: "invalid data encountered in the Redis config form",
	Fields: []forms.Field{
		{
			Name: "connection_string",
			Validators: []forms.Validator{
				forms.IsString{},
			},
		},
	},
}

func ValidatePostgeSQLSettings(settings map[string]interface{}) (interface{}, error) {
	if params, err := PostgreSQLForm.Validate(settings); err != nil {
		return nil, err
	} else {
		pgSettings := &PostgreSQLSettings{}
		if err := PostgreSQLForm.Coerce(pgSettings, params); err != nil {
			return nil, err
		}
		return pgSettings, nil
	}
}

func MakePostgreSQL(settings interface{}) (services.Database, error) {
	pgSettings := settings.(PostgreSQLSettings)
	services.Log.Info("Connecting to PostgreSQL DBMS")
	ctx := context.Background()
	pool, err := pg.Connect (ctx, pgSettings.ConnString)
	if err != nil {
		return nil, err
	} else {
		return &PostgreSQL{pool: pool, ctx: ctx}, nil
	}
}

var _ services.Database = &PostgreSQL{}

func (d *PostgreSQL) Reset() error {
	return nil
}

func (d *PostgreSQL) Close() error {
	services.Log.Info("Closing connection to PostgreSQL DBMS")
	d.pool.Close()
	return nil
}

func (d *PostgreSQL) Open() error {
	return nil
}

func (d *PostgreSQL) SettingsDelete (id string) error {
	sqlStr := `DELETE FROM "storage" WHERE "storage_id" = $1`
	_, err := d.pool.Exec(d.ctx, sqlStr, id)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

func (d *PostgreSQL) SettingsGet (id string) ([]byte, error) {
	sqlStr := `
		UPDATE "storage" SET "accessed_at" = NOW() WHERE "storage_id" = $1
			RETURNING "data"
	`
	var data []byte
	if err := d.pool.QueryRow(d.ctx, sqlStr, id).Scan(&data); err != nil {
		if err == pgx.ErrNoRows {
			return nil, NotFound
		} else {
			services.Log.Debug("psql query failed: ", err)
			return nil, err
		}
	}

	return data, nil
}

func (d *PostgreSQL) SettingsReset () error {
	sqlStr := `DELETE FROM "storage"`
	_, err := d.pool.Exec(d.ctx, sqlStr)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

func (d *PostgreSQL) SettingsStore (id string, data []byte) error {
	sqlStr := `
		INSERT INTO "storage" ("storage_id", "data") VALUES ($1, $2)
			ON CONFLICT ("storage_id") DO UPDATE 
			SET data = EXCLUDED."data", "updated_at" = NOW(), "accessed_at" = NOW()
	`
	_, err := d.pool.Exec(d.ctx, sqlStr, id, data)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

