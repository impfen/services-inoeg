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

package servers_test

import (
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/definitions"
	"github.com/impfen/services-inoeg/forms"
	"github.com/impfen/services-inoeg/helpers"
	at "github.com/impfen/services-inoeg/testing"
	af "github.com/impfen/services-inoeg/testing/fixtures"
	"testing"
)

func TestConfirmProvider(t *testing.T) {

	var fixturesConfig = []at.FC{

		// we create the settings
		at.FC{af.Settings{Definitions: definitions.Default}, "settings"},

		// we create the appointments API
		at.FC{af.AppointmentsServer{}, "appointmentsServer"},

		// we create a client (without a key)
		at.FC{af.Client{}, "client"},

		// we create a mediator
		at.FC{af.Mediator{}, "mediator"},

		// we create a mediator
		at.FC{af.Provider{
			ZipCode:   "10707",
			StoreData: true,
			Confirm:   true,
		}, "provider"},
	}

	fixtures, err := at.SetupFixtures(fixturesConfig)
	defer at.TeardownFixtures(fixturesConfig, fixtures)

	if err != nil {
		t.Fatal(err)
	}

	client := fixtures["client"].(*helpers.Client)
	provider := fixtures["provider"].(*helpers.Provider)

	resp, err := client.Appointments.CheckProviderData(provider)

	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("expected a 200 status code, got %d instead", resp.StatusCode)
	}

	result := &services.ConfirmedProviderData{}

	if err := resp.CoerceResult(result, &forms.ConfirmedProviderDataForm); err != nil {
		t.Fatal(err)
	}

}
