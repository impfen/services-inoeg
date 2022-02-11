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

package servers

import (
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/api"
	"github.com/impfen/services-inoeg/forms"
)

type Storage struct {
	*Server
	settings *services.StorageSettings
	db       services.Database
	test     bool
}

func MakeStorage(settings *services.Settings) (*Storage, error) {

	storage := &Storage{
		db:       settings.DatabaseObj,
		settings: settings.Storage,
		test:     settings.Test,
	}

	api := &api.API{
		Version: 1,
		Name:    "storage",
		Endpoints: []*api.Endpoint{
			{
				Name:        "storeSettings",
				Description: "Stores encrypted settings.",
				Form:        &forms.StoreSettingsForm,
				Handler:     storage.storeSettings,
				ReturnType: &api.ReturnType{
					Validators: forms.IsAcknowledgeRVV,
				},
				REST: &api.REST{
					Path:   "store",
					Method: api.PUT,
				},
			},
			{
				Name:        "getSettings",
				Description: "Retrieves encrypted settings.",
				Form:        &forms.GetSettingsForm,
				Handler:     storage.getSettings,
				REST: &api.REST{
					Path:   "store/<id>",
					Method: api.GET,
				},
			},
			{
				Name:        "deleteSettings",
				Description: "Deletes encrypted settings.",
				Form:        &forms.DeleteSettingsForm,
				Handler:     storage.deleteSettings,
				ReturnType: &api.ReturnType{
					Validators: forms.IsAcknowledgeRVV,
				},
				REST: &api.REST{
					Path:   "store",
					Method: api.DELETE,
				},
			},
			{
				Name:        "resetDB",
				Description: "Resets the database. Only enabled for test deployments.",
				Form:        &forms.ResetDBForm,
				Handler:     storage.resetDB,
				ReturnType: &api.ReturnType{
					Validators: forms.IsAcknowledgeRVV,
				},
				REST: &api.REST{
					Path:   "db/reset",
					Method: api.DELETE,
				},
			},
		},
	}

	var err error

	if storage.Server, err = MakeServer("storage", settings.Storage.HTTP, settings.Storage.JSONRPC, settings.Storage.REST, settings.Appointments.Validate, api); err != nil {
		return nil, err
	}

	return storage, nil

}

func (c *Storage) isRoot(context services.Context, params *services.SignedParams) services.Response {
	return isRoot(context, []byte(params.JSON), params.Signature, params.Timestamp, c.settings.Keys)
}
