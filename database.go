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

package services

import (
	"time"
)

type DatabaseDefinition struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Maker             DatabaseMaker     `json:"-"`
	SettingsValidator SettingsValidator `json:"-"`
}

type SettingsValidator func(settings map[string]interface{}) (interface{}, error)
type DatabaseDefinitions map[string]DatabaseDefinition
type DatabaseMaker func(settings interface{}) (Database, error)

type DatabaseOps interface {
	AppointmentsReset() error
	AppointmentUpsert([]byte, []*SignedAppointment) error
	AppointmentsGetByDateRange([]byte, time.Time, time.Time) ([]*SignedAppointment, error)
	MediatorKeysGetAll() ([]*ActorKey, error)
	MediatorUpsert(key *ActorKey) error
	ProviderGetByID([]byte) (*SqlProvider, error)
	ProviderGetAll(string) ([]*SqlProvider, error)
	ProviderGetPublicByZip(string, string) ([]*SqlProvider, error)
	ProviderKeysGetAll() ([]*ActorKey, error)
	ProviderPublishData(*RawProviderData) error
	ProviderVerify(*ActorKey, *ConfirmedProviderData, *SignedProviderData) error
	SettingsDelete(id string) error
	SettingsGet(id string) ([]byte, error)
	SettingsReset() error
	SettingsUpsert(id string, data []byte) error
	TokenPrimaryAdd(int64) (int64, error)
	TokenUserAdd([]byte, int64) (int64, error)
}

// A database can deliver and accept message
type Database interface {
	Close() error
	DatabaseOps
}

type SqlProvider struct {
	ID             []byte                 `json:"id"`
	Name           string                 `json:"name"`
	Street         string                 `json:"street"`
	City           string                 `json:"city"`
	ZipCode        string                 `json:"zip_code"`
	Description    string                 `json:"description"`
	Accessible     bool                   `json:"accessible"`
	KeyData        string                 `json:"key_data"`
	KeySignature   []byte                 `json:"key_signature"`
	PublicKey      []byte                 `json:"public_key"`
	Active         bool                   `json:"active"`
	UnverifiedData *RawProviderData       `json:"unverified_data"`
	VerifiedData   *RawProviderData       `json:"verified_data"`
	ConfirmedData  *ConfirmedProviderData `json:"confirmed_data"`
	PublicData     *SignedProviderData    `json:"public_data"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
