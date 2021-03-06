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
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/databases"
	"time"
)

func toInterface(data []byte) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (c *Storage) getSettings(
	context services.Context,
	params *services.GetSettingsParams,
) services.Response {

	value := c.db.Value("settings", params.ID)

	if data, err := value.Get(); err != nil {
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
		ttl := time.Duration(c.settings.SettingsTTLDays*24)*time.Hour
		c.db.Expire("settings", params.ID, ttl)
		return context.Result(i)
	}
}
