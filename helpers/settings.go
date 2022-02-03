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

// This code was extracted from the Kodex CE project
// (https://github.com/kiprotect/kodex), the author has
// the copyright to the code so no attribution is necessary.

package helpers

import (
	"github.com/impfen/services-inoeg"
	servicesForms "github.com/impfen/services-inoeg/forms"
	"github.com/kiprotect/go-helpers/forms"
	"github.com/kiprotect/go-helpers/settings"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var EnvSettingsName = "KIEBITZ_SETTINGS"

func SettingsPaths() ([]string, fs.FS, error) {
	if paths, err := RealSettingsPaths(); err != nil {
		return nil, nil, err
	} else {
		modifiedPaths := make([]string, len(paths))
		for i, path := range paths {
			modifiedPaths[i] = path[1:]
		}
		return modifiedPaths, os.DirFS("/"), nil
	}
}

func RealSettingsPaths() ([]string, error) {
	envValue := os.Getenv(EnvSettingsName)
	values := strings.Split(envValue, ":")
	sanitizedValues := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		var err error
		if value, err = filepath.Abs(value); err != nil {
			return nil, err
		}
		sanitizedValues = append(sanitizedValues, value)
	}
	return sanitizedValues, nil
}

func Settings(settingsPaths []string, fs fs.FS, definitions *services.Definitions) (*services.Settings, error) {
	if rawSettings, err := settings.MakeSettings(settingsPaths, fs); err != nil {
		return nil, err
	} else if params, err := servicesForms.SettingsForm.ValidateWithContext(rawSettings.Values, map[string]interface{}{"definitions": definitions}); err != nil {
		return nil, err
	} else {
		settings := &services.Settings{
			Definitions: definitions,
		}
		if err := forms.Coerce(settings, params); err != nil {
			// this should not happen if the forms are correct...
			return nil, err
		}
		// settings are valid
		return settings, nil
	}
}
