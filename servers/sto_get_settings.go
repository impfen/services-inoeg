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
	"encoding/json"
	"github.com/kiebitz-oss/services"
	"github.com/kiebitz-oss/services/databases"
)

func toInterface(data []byte) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Storage) getSettings(
	context services.Context,
	params *services.GetSettingsParams,
) services.Response {

	if data, err := s.backend.getSettings(params.ID); err != nil {
		if err == databases.NotFound {
			return context.NotFound()
		} else {
			services.Log.Error(err)
			return context.InternalError()
		}

	} else if i, err := toInterface(data); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	} else {
		return context.Result(i)
	}
}
