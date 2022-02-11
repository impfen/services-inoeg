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

package forms

import (
	"fmt"
	"github.com/impfen/services-inoeg"
)

type AreValidDatabaseSettings struct {
}

func (f AreValidDatabaseSettings) Validate(input interface{}, inputs map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("cannot validate without context")
}

func (f AreValidDatabaseSettings) ValidateWithContext(input interface{}, inputs map[string]interface{}, context map[string]interface{}) (interface{}, error) {
	definitions, ok := context["definitions"].(*services.Definitions)
	if !ok {
		return nil, fmt.Errorf("expected a 'definitions' context")
	}
	databaseType := inputs["type"].(string)
	// string type has been validated before
	settings := input.(map[string]interface{})
	if definition, ok := definitions.DatabaseDefinitions[databaseType]; !ok {
		return nil, fmt.Errorf("invalid database type: '%s'", databaseType)
	} else if definition.SettingsValidator == nil {
		return nil, fmt.Errorf("cannot validate settings for database of type '%s'", databaseType)
	} else if validatedSettings, err := definition.SettingsValidator(settings); err != nil {
		return nil, err
	} else {
		return validatedSettings, nil
	}
}

type IsValidDatabaseType struct {
}

func (f IsValidDatabaseType) Validate(input interface{}, inputs map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("cannot validate without context")
}

func (f IsValidDatabaseType) ValidateWithContext(input interface{}, inputs map[string]interface{}, context map[string]interface{}) (interface{}, error) {
	definitions, ok := context["definitions"].(*services.Definitions)
	if !ok {
		return nil, fmt.Errorf("expected a 'definitions' context")
	}
	// string type has been validated before
	strValue := input.(string)
	if _, ok := definitions.DatabaseDefinitions[strValue]; !ok {
		return nil, fmt.Errorf("invalid database type: '%s'", strValue)
	}
	return input, nil
}
