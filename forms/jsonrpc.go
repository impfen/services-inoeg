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
	"github.com/kiebitz-oss/services/tls"
	"github.com/kiprotect/go-helpers/forms"
	"regexp"
)

type IsValidRegexp struct {}

func (i IsValidRegexp) Validate(
	value interface{},
	values map[string]interface{},
) (interface{}, error) {
	// Assumes that stringiness has been validated before...
	if _, err := regexp.Compile(value.(string)); err != nil {
		return nil, err
	}
	return value, nil
}

var CorsSettingsForm = forms.Form{
	Name:     "corsSettings",
	ErrorMsg: "invalid data encountered in the CORS settings form",
	Fields: []forms.Field{
		{
			Name: "allowed_hosts",
			Validators: []forms.Validator{
				forms.IsOptional{Default: []string{}},
				forms.IsStringList{
					Validators: []forms.Validator{
						IsValidRegexp{},
					},
				},
			},
		},
		{
			Name: "allowed_headers",
			Validators: []forms.Validator{
				forms.IsOptional{Default: []string{}},
				forms.IsStringList{},
			},
		},
		{
			Name: "allowed_methods",
			Validators: []forms.Validator{
				forms.IsOptional{Default: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}},
				forms.IsList{
					Validators: []forms.Validator{
						forms.IsIn{
							Choices: []interface{}{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
						},
					},
				},
				forms.IsStringList{},
			},
		},
	},
}

var JSONRPCServerSettingsForm = forms.Form{
	Fields: []forms.Field{
		{
			Name: "cors",
			Validators: []forms.Validator{
				forms.IsOptional{},
				forms.IsStringMap{
					Form: &CorsSettingsForm,
				},
			},
		},
	},
}

var JSONRPCClientSettingsForm = forms.Form{
	Fields: []forms.Field{
		{
			Name: "endpoint",
			Validators: []forms.Validator{
				forms.IsString{}, // to do: add URL validation
			},
		},
		{
			Name: "local",
			Validators: []forms.Validator{
				forms.IsOptional{Default: true},
				forms.IsBoolean{},
			},
		},
		{
			Name: "tls",
			Validators: []forms.Validator{
				forms.IsOptional{},
				forms.IsStringMap{
					Form: &tls.TLSSettingsForm,
				},
			},
		},
	},
}
