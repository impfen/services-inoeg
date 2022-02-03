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
	"encoding/base64"
	pg "github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4"
	"github.com/kiebitz-oss/services"
	"github.com/kiprotect/go-helpers/forms"
	"time"
)

func toBase64 (bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func fromBase64 (str string) []byte {
	if res, err := base64.StdEncoding.DecodeString(str); err != nil {
		return []byte{}
	} else {
		return res
	}
}

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

func (d *PostgreSQL) Close() error {
	services.Log.Info("Closing connection to PostgreSQL DBMS")
	d.pool.Close()
	return nil
}

func (d *PostgreSQL) AppointmentsReset () error {
	sqlStr := `DELETE FROM "mediator"; DELETE FROM "provider"`
	_, err := d.pool.Exec(d.ctx, sqlStr)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

func (d *PostgreSQL) MediatorKeysGetAll () ([]*services.ActorKey, error) {
	sqlStr := `
		SELECT "mediator_id", "key_data", "key_signature", "public_key"
			FROM "mediator"
			WHERE active
	`
	res, err := d.pool.Query(d.ctx, sqlStr)
	defer res.Close()
	if err != nil {
		services.Log.Debug("psql query failed: ", err)
		return nil, err
	}

	ms := []*services.ActorKey{}
	for res.Next() {
		var id, data string
		var sig, key []byte
		res.Scan(&id, &data, &sig, &key)
		ms = append(ms, &services.ActorKey{
			ID:        fromBase64(id),
			Data:      data,
			Signature: sig,
			PublicKey: key,
		})
	}
	return ms, nil
}

func (d *PostgreSQL) MediatorUpsert (key *services.ActorKey) error {
	sqlStr := `
		INSERT INTO "mediator"
			("mediator_id", "key_data", "key_signature", "public_key")
			VALUES ($1, $2, $3, $4)
			ON CONFLICT ("mediator_id") DO UPDATE 
			SET "key_data" = EXCLUDED."key_data"
				, "key_signature" = EXCLUDED."key_signature"
				, "public_key" = EXCLUDED."public_key"
				, "updated_at" = NOW()
	`
	_, err := d.pool.Exec(d.ctx, sqlStr,
		toBase64(key.ID), // $1
		key.Data,         // $2
		key.Signature,    // $3
		key.PublicKey,    // $4
	)

	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

func rowToSqlProvider (row pgx.Row) (*services.SqlProvider, error) {

	var id, name, street, city, zipCode, description, keyData string
	var accessible, active bool
	var keySignature, publicKey []byte
	var createdAt, updatedAt time.Time
	var unverifiedData, verifiedData *services.RawProviderData
	var confirmedData *services.ConfirmedProviderData
	var publicData *services.SignedProviderData

	err := row.Scan(
		&id,
		&name,
		&street,
		&city,
		&zipCode,
		&description,
		&accessible,
		&keyData,
		&keySignature,
		&publicKey,
		&active,
		&unverifiedData,
		&verifiedData,
		&confirmedData,
		&publicData,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		services.Log.Debug("psql query failed: ", err)
		return nil, err
	}

	return &services.SqlProvider{
		ID:             fromBase64(id),
		Name:           name,
		Street:         street,
		City:           city,
		ZipCode:        zipCode,
		Description:    description,
		Accessible:     accessible,
		KeyData:        keyData,
		KeySignature:   keySignature,
		PublicKey:      publicKey,
		Active:         active,
		UnverifiedData: unverifiedData,
		VerifiedData:   verifiedData,
		ConfirmedData:  confirmedData,
		PublicData:     publicData,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

var providerRows = `
  "provider_id",
  "name",
  "street",
  "city",
  "zip_code",
  "description",
  "accessible",
  "key_data",
  "key_signature",
  "public_key",
  "active",
  "unverified_data",
  "verified_data",
  "confirmed_data",
  "public_data",
  "created_at",
  "updated_at"
`

func (d *PostgreSQL) ProviderGetByID(
	providerID []byte,
) (*services.SqlProvider, error) {
	sqlStr := `SELECT ` + providerRows + `FROM "provider" WHERE "provider_id" = $1`

	row := d.pool.QueryRow(d.ctx, sqlStr, toBase64(providerID))
	if provider, err := rowToSqlProvider(row) ; err != nil {
		if err == pgx.ErrNoRows {
			return nil, NotFound
		} else {
			services.Log.Debug("psql query failed: ", err)
			return nil, err
		}
	} else {
		return provider, nil
	}
}

func (d *PostgreSQL) ProviderGetAll(
	filter string,
) ([]*services.SqlProvider, error) {
	sqlFilter := ""
	switch filter {
		case "verified":
			sqlFilter = ` WHERE "verified_data" IS NOT NULL `
		case "unverified":
			sqlFilter = ` WHERE "unverified_data" IS NOT NULL `
	}
	sqlStr := `SELECT ` + providerRows + ` FROM "provider" ` + sqlFilter + ` ORDER BY "provider_id"`

	res, err := d.pool.Query(d.ctx, sqlStr)
	defer res.Close()
	if err != nil {
		services.Log.Debug("psql query failed: ", err)
		return nil, err
	}

	providers := []*services.SqlProvider{}
	for res.Next() {
		p, err := rowToSqlProvider(res)
		providers = append(providers, p)
		if err != nil {
			services.Log.Debug("psql query failed: ", err)
			return nil, err
		}
	}

	return providers, nil
}

func (d *PostgreSQL) ProviderKeysGetAll () ([]*services.ActorKey, error) {
	sqlStr := `
		SELECT "provider_id", "key_data", "key_signature", "public_key"
			FROM "provider"
			WHERE active
	`
	res, err := d.pool.Query(d.ctx, sqlStr)
	defer res.Close()
	if err != nil {
		services.Log.Debug("psql query failed: ", err)
		return nil, err
	}

	ms := []*services.ActorKey{}
	for res.Next() {
		var id, data string
		var sig, key []byte
		res.Scan(&id, &data, &sig, &key)
		ms = append(ms, &services.ActorKey{
			ID:        fromBase64(id),
			Data:      data,
			Signature: sig,
			PublicKey: key,
		})
	}
	return ms, nil
}

func (d *PostgreSQL) ProviderPublishData (
	data *services.RawProviderData,
) error {
	sqlStr := `
		INSERT INTO "provider" ("provider_id", "unverified_data") VALUES ($1, $2)
			ON CONFLICT ("provider_id") DO UPDATE 
			SET "unverified_data" = EXCLUDED."unverified_data", "updated_at" = NOW()
	`
	_, err := d.pool.Exec(d.ctx, sqlStr, toBase64(data.ID), data)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

func (d *PostgreSQL) ProviderVerify (
	key *services.ActorKey,
	confirmedData *services.ConfirmedProviderData,
	publicData *services.SignedProviderData,
) error {
	sqlStr := `
		UPDATE "provider"
			SET "verified_data" = "unverified_data"
				, "unverified_data" = NULL
				, "confirmed_data" = $2
				, "public_data" = $3
				, "name" = $4
				, "street" = $5
				, "city" = $6
				, "zip_code" = $7
				, "description" = $8
				, "accessible" = $9
				, "key_data" = $10
				, "key_signature" = $11
				, "public_key" = $12
				, "active" = true
				, "updated_at" = NOW()
			WHERE "provider_id" = $1
	`
	_, err := d.pool.Exec(d.ctx, sqlStr,
		toBase64(key.ID),            //  $1
		confirmedData,               //  $2
		publicData,                  //  $3
		publicData.Data.Name,        //  $4
		publicData.Data.Street,      //  $5
		publicData.Data.City,        //  $6
		publicData.Data.ZipCode,     //  $7
		publicData.Data.Description, //  $8
		publicData.Data.Accessible,  //  $9
		key.Data,                    // $10
		key.Signature,               // $11
		key.PublicKey,               // $12
	)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
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

func (d *PostgreSQL) SettingsUpsert (id string, data []byte) error {
	sqlStr := `
		INSERT INTO "storage" ("storage_id", "data") VALUES ($1, $2)
			ON CONFLICT ("storage_id") DO UPDATE 
			SET data = EXCLUDED."data", "updated_at" = NOW(), "accessed_at" = NOW()
	`
	_, err := d.pool.Exec(d.ctx, sqlStr, id, data)
	if err != nil { services.Log.Debug("psql query failed: ", err) }
	return err
}

