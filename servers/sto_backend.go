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
	"github.com/kiebitz-oss/services"
)

type StorageBackend struct {
	db services.Database
}

func (s *StorageBackend) deleteSettings (id []byte) error {
	return s.db.SettingsDelete(toBase64(id))
}

func (s *StorageBackend) getSettings (id []byte) ([]byte, error) {
	return s.db.SettingsGet(toBase64(id))
}

func (s *StorageBackend) reset () error {
	return s.db.SettingsReset()
}

func (s *StorageBackend) storeSettings (id, data []byte) error {
	return s.db.SettingsUpsert(toBase64(id), data)
}
