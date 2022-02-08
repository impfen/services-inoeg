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
	"github.com/kiebitz-oss/services/crypto"
	"github.com/kiprotect/go-helpers/forms"
	"math"
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
	ErrorMsg: "invalid data encountered in the PostgrePostgres config form",
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
	sqlStr := `
		DELETE FROM "slot";
		DELETE FROM "property";
		DELETE FROM "appointment";
		DELETE FROM "provider";
		DELETE FROM "mediator";
	`
	_, err := d.pool.Exec(d.ctx, sqlStr)
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
	return err
}

func (d *PostgreSQL) AppointmentBook (
	providerID []byte,
	appointmentID []byte,
	publicKey []byte,
	token []byte,
	encryptedData *crypto.ECDHEncryptedData,
) ([]byte, error) {

	insertTokenSqlStr := `INSERT INTO "used_token" ("token_id") VALUES ($1)`
	getSlotSqlStr := `
		SELECT "slot_id"
		FROM "slot"
		WHERE "appointment" = $1 AND "token" IS NULL
		FOR UPDATE
	`
	updateSlotSqlStr := `
		UPDATE "slot"
		SET "token" = $1, "public_key" = $2, "encrypted_data" = $3
		WHERE "slot_id" = $4
	`

	tx, err := d.pool.Begin(d.ctx)
	if err != nil {
		services.Log.Debugf("can't begin transaction: ", err)
		return nil, err
	}
	defer tx.Rollback(d.ctx)

	_, err = tx.Exec(d.ctx, insertTokenSqlStr, toBase64(token))
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, ErrTokenUsed
	}

	var openSlot string
	err = tx.QueryRow(
		d.ctx,
		getSlotSqlStr,
		toBase64(appointmentID),
	).Scan(&openSlot)
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, NotFound
	}

	_, err = tx.Exec(
		d.ctx,
		updateSlotSqlStr,
		toBase64(token),
		publicKey,
		encryptedData,
		openSlot,
	)
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	tx.Commit(d.ctx)
	return fromBase64(openSlot), err
}

func (d *PostgreSQL) AppointmentCancel (
	appointmentID []byte,
	token []byte,
) error {
	var delBookingSqlStr = `
		UPDATE "slot"
		SET "token" = NULL, "public_key" = NULL, "encrypted_data" = NULL
		WHERE "appointment" = $1 AND "token" = $2
	`
	var delTokenSqlStr = `DELETE FROM "used_token" WHERE "token_id" = $1`

	_, err := d.pool.Exec(d.ctx, delBookingSqlStr,
		toBase64(appointmentID),
		toBase64(token),
	)
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return err
	}

	_, err = d.pool.Exec(d.ctx, delTokenSqlStr, toBase64(token))
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return err
	}

	return nil
}

func (d *PostgreSQL) rowToSignedAppointment (
	row pgx.Row,
) (*services.SignedAppointment, error){
	var getSlotSqlStr = `
		SELECT "slot_id", "token", "public_key", "encrypted_data"
		FROM "slot"
		WHERE "appointment" = $1 AND "slot"."token" IS NOT NULL
	`
	
	var appId string
	app := &services.SignedAppointment{}
	app.Bookings = []*services.Booking{}

	row.Scan(&appId, &app.JSON, &app.Signature, &app.PublicKey, &app.UpdatedAt)

	rowsSlots, err := d.pool.Query(d.ctx, getSlotSqlStr, appId)
	defer rowsSlots.Close()
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	for rowsSlots.Next() {
		var slotID string
		slot := &services.Booking{}
		rowsSlots.Scan(&slotID, &slot.PublicKey, &slot.Token, &slot.EncryptedData)
		slot.ID = fromBase64(slotID)
		app.Bookings = append(app.Bookings, slot)
	}

	return app, nil
}

func (d *PostgreSQL) AppointmentGet (
	appointmentID []byte,
	providerID []byte,
) (*services.SignedAppointment, error) {
	var getAppSqlStr = `
		SELECT
			  "appointment_id"
			, "signed_data"
			, "signature"
			, "public_key"
			, "updated_at"
		FROM "appointment"
		WHERE "appointment_id" = $1 AND "provider" = $2
	`

	row := d.pool.QueryRow(
		d.ctx,
		getAppSqlStr,
		toBase64(appointmentID),
		toBase64(providerID),
	)

	app, err := d.rowToSignedAppointment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, NotFound
		} else {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	return app, nil
}
func (d *PostgreSQL) AppointmentsGetByProperty (
	providerID []byte,
	key string,
	val string,
) ([]*services.SignedAppointment, error) {
	var getAppSqlStr = `
		SELECT
			  "appointment"."appointment_id"
			, "appointment"."signed_data"
			, "appointment"."signature"
			, "appointment"."public_key"
			, "appointment"."updated_at"
		FROM "appointment", "property"
		WHERE "appointment"."provider" = $1
			AND "appointment"."appointment_id" = "property"."appointment"
			AND "property"."key" = $2
			AND "property"."value" = $3
		ORDER BY "appointment"."timestamp"
	`

	appointments := []*services.SignedAppointment{}

	rows, err := d.pool.Query(d.ctx, getAppSqlStr, toBase64(providerID), key, val)
	defer rows.Close()
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	for rows.Next() {
		app, err := d.rowToSignedAppointment(rows)
		if err != nil { return nil, err }
		appointments = append(appointments, app)
	}

	return appointments, nil
}

func (d *PostgreSQL) AppointmentsGetByDateRange (
	providerID []byte,
	dateFrom time.Time,
	dateTo time.Time,
) ([]*services.SignedAppointment, error) {
	var getAppSqlStr = `
		SELECT
			  "appointment_id"
			, "signed_data"
			, "signature"
			, "public_key"
			, "updated_at"
		FROM "appointment"
		WHERE "provider" = $1
			AND "timestamp" BETWEEN $2 AND $3
		ORDER BY "appointment"."timestamp"
	`

	appointments := []*services.SignedAppointment{}

	rows, err := d.pool.Query(d.ctx, getAppSqlStr, toBase64(providerID), dateFrom, dateTo)
	defer rows.Close()
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	for rows.Next() {
		app, err := d.rowToSignedAppointment(rows)
		if err != nil { return nil, err }
		appointments = append(appointments, app)
	}

	return appointments, nil

}

func (d *PostgreSQL) AppointmentUpsert (
	providerID []byte,
	appointments []*services.SignedAppointment,
) error {
	insertAppSqlStr := `
		INSERT INTO "appointment"
			( "appointment_id"
			, "provider"
			, "duration"
			, "timestamp"
			, "vaccine"
			, "signed_data"
			, "signature"
			, "public_key"
			)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT ("appointment_id") DO UPDATE 
			SET "timestamp" = EXCLUDED."timestamp"
				, "duration" = EXCLUDED."duration"
				, "vaccine" = EXCLUDED."vaccine"
				, "signed_data" = EXCLUDED."signed_data"
				, "signature" = EXCLUDED."signature"
				, "public_key" = EXCLUDED."public_key"
				, "updated_at" = NOW()
	`
	insertSlotSqlStr := `
		INSERT INTO "slot" ("slot_id" ,"appointment")
		VALUES ($1, $2)
		ON CONFLICT ("slot_id") DO NOTHING
	`
	
	deleteSlotSqlStr := `
		DELETE FROM "slot"
		WHERE "appointment" = $1 AND "slot_id" <> ALL ($2)
	`

	insertPropertySqlStr := `
		INSERT INTO "property" ("key", "value" ,"appointment")
		VALUES ($1, $2, $3)
	`
	
	deletePropertySqlStr := `DELETE FROM "property" WHERE "appointment" = $1`

	var err error

	tx, err := d.pool.Begin(d.ctx)
	if err != nil {
		services.Log.Debugf("can't begin transaction: ", err)
		return err
	}
	defer tx.Rollback(d.ctx)

	for _, appointment := range appointments {
		slotIDs := []string{}
		for _, slot := range appointment.Data.SlotData {
			slotIDs = append(slotIDs, toBase64(slot.ID))
		}

		// delete old data
		_, err = tx.Exec(d.ctx, deleteSlotSqlStr,
			toBase64(appointment.Data.ID), // $1
			slotIDs,                       // $2
		)
		if err != nil {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return err
		}

		_, err = tx.Exec(d.ctx, deletePropertySqlStr,
			toBase64(appointment.Data.ID), // $1
		)
		if err != nil {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return err
		}

		// insert new data
		_, err = tx.Exec(d.ctx, insertAppSqlStr,
			toBase64(appointment.Data.ID), // $1
			toBase64(providerID),          // $2
			appointment.Data.Duration,     // $3
			appointment.Data.Timestamp,    // $4
			appointment.Data.Vaccine,      // $5
			appointment.JSON,              // $6
			appointment.Signature,         // $7
			appointment.PublicKey,         // $8
		)
		if err != nil {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return err
		}

		for _, slot := range appointment.Data.SlotData {
			_, err := tx.Exec(d.ctx, insertSlotSqlStr,
				toBase64(slot.ID),             // $1
				toBase64(appointment.Data.ID), // $2
			)
			if err != nil {
				services.Log.Debugf("psql query failed:\n%#v\n", err)
				return err
			}
		}

		for key, val := range appointment.Data.Properties {
			_, err := tx.Exec(d.ctx, insertPropertySqlStr,
				key,                           // $1
				val,                           // $2
				toBase64(appointment.Data.ID), // $3
			)
			if err != nil {
				services.Log.Debugf("psql query failed:\n%#v\n", err)
				return err
			}
		}
	}

	tx.Commit(d.ctx)
	return nil
}

func (d *PostgreSQL) MediatorKeyFind (id []byte) (*services.ActorKey, error) {
	sqlStr := `
		SELECT "mediator_id", "key_data", "key_signature", "public_key"
			FROM "mediator"
			WHERE active
				AND (
					"mediator_id" = $1
					OR "key_data"::jsonb->'signing' = to_jsonb($1)
				)
	`

	mediatorKey := &services.ActorKey{}
	var mediatorID string

	row := d.pool.QueryRow(d.ctx, sqlStr, toBase64(id))
	err := row.Scan(
		&mediatorID,
		&mediatorKey.Data,
		&mediatorKey.Signature,
		&mediatorKey.PublicKey,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, NotFound
		} else {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	mediatorKey.ID = fromBase64(mediatorID)

	return mediatorKey, nil
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
		services.Log.Debugf("psql query failed:\n%#v\n", err)
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

	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
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
		services.Log.Debugf("psql query failed:\n%#v\n", err)
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
			services.Log.Debugf("psql query failed:\n%#v\n", err)
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
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	providers := []*services.SqlProvider{}
	for res.Next() {
		p, err := rowToSqlProvider(res)
		providers = append(providers, p)
		if err != nil {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	return providers, nil
}

func (d *PostgreSQL) ProviderGetPublicByZip(
	zipFrom string,
	zipTo string,
) ([]*services.SqlProvider, error) {
	sqlStr := `
		SELECT ` + providerRows + `
		FROM "provider"
		WHERE active AND zip_code >= $1 AND zip_code <= $2
		ORDER BY "provider_id"
	`

	res, err := d.pool.Query(d.ctx, sqlStr, zipFrom, zipTo)
	defer res.Close()
	if err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return nil, err
	}

	providers := []*services.SqlProvider{}
	for res.Next() {
		p, err := rowToSqlProvider(res)
		providers = append(providers, p)
		if err != nil {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	return providers, nil
}

func (d *PostgreSQL) ProviderKeyGetByID (
	providerID []byte,
) (*services.ActorKey, error) {
	sqlStr := `
		SELECT "provider_id", "key_data", "key_signature", "public_key"
			FROM "provider"
			WHERE "provider_id" = $1
	`

	providerKey := &services.ActorKey{}
	var id string

	row := d.pool.QueryRow(d.ctx, sqlStr, toBase64(providerID))
	err := row.Scan(
		&id,
		&providerKey.Data,
		&providerKey.Signature,
		&providerKey.PublicKey,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, NotFound
		} else {
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	providerKey.ID = fromBase64(id)

	return providerKey, nil
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
		services.Log.Debugf("psql query failed:\n%#v\n", err)
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
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
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
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
	return err
}


func (d *PostgreSQL) SettingsDelete (id string) error {
	sqlStr := `DELETE FROM "storage" WHERE "storage_id" = $1`
	_, err := d.pool.Exec(d.ctx, sqlStr, id)
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
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
			services.Log.Debugf("psql query failed:\n%#v\n", err)
			return nil, err
		}
	}

	return data, nil
}

func (d *PostgreSQL) SettingsReset () error {
	sqlStr := `DELETE FROM "storage"`
	_, err := d.pool.Exec(d.ctx, sqlStr)
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
	return err
}

func (d *PostgreSQL) SettingsUpsert (id string, data []byte) error {
	sqlStr := `
		INSERT INTO "storage" ("storage_id", "data") VALUES ($1, $2)
			ON CONFLICT ("storage_id") DO UPDATE 
			SET data = EXCLUDED."data", "updated_at" = NOW(), "accessed_at" = NOW()
	`
	_, err := d.pool.Exec(d.ctx, sqlStr, id, data)
	if err != nil { services.Log.Debugf("psql query failed:\n%#v\n", err) }
	return err
}

func (d *PostgreSQL) TokenPrimaryAdd (n int64) (int64, error) {
	sqlStr := `
		UPDATE "token" SET "n" = "n" + $1 WHERE "name" = 'primary'
		RETURNING "n"
	`

	var newN int64

	if err := d.pool.QueryRow(d.ctx, sqlStr, n).Scan(&newN); err != nil {
		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return math.MaxInt64, err
	}

	return newN, nil
}

func (d *PostgreSQL) TokenUserAdd (userID []byte, n int64) (int64, error) {
	sqlStr := `
		INSERT INTO "user_token" ("user_id", "n") VALUES ($1, $2)
			ON CONFLICT ("user_id") DO UPDATE
			SET "n" = "user_token"."n" + EXCLUDED."n"
			RETURNING "n"
	`

	var newN int64

	if err := d.pool.QueryRow(
		d.ctx,
		sqlStr,
		toBase64(userID),
		n,
	).Scan(&newN); err != nil {

		services.Log.Debugf("psql query failed:\n%#v\n", err)
		return math.MaxInt64, err
	}

	return newN, nil
}
