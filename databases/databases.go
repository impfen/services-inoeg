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
	"github.com/kiebitz-oss/services"
)

var Databases = services.DatabaseDefinitions{
	"postgresql": services.DatabaseDefinition{
		Name:              "PostgreSQL database",
		Description:       "realtional database for production use",
		Maker:             MakePostgreSQL,
		SettingsValidator: ValidatePostgeSQLSettings,
	},
	/* TODO
	"sqlite": services.DatabaseDefinition{
		Name:              "SQLite",
		Description:       "A light weight dbms which can be run in memory",
		Maker:             MakeSQLite,
		SettingsValidator: ValidateSQLiteSettings,
	},
	*/
}
