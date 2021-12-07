// Kiebitz - Privacy-Friendly Appointment Scheduling
// Copyright (C) 2021-2021 The Kiebitz Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
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
)

var HTTPServerSettingsForm = forms.Form{
	Fields: []forms.Field{
		{
			Name: "bind_address",
			Validators: []forms.Validator{
				forms.IsString{}, // to do: add URL validation
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