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
	"encoding/base64"
	"github.com/kiebitz-oss/services"
)

func toBase64 (bytes []byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func (c *Appointments) LockProvider (
	providerId []byte,
) (services.Lock, error) {

	return c.db.LockDefault(
		"Lock::Provider::" + toBase64(providerId),
	)
}

func (c *Appointments) LockAppointment (
	appointmentId []byte,
) (services.Lock, error) {

	return c.db.LockDefault(
		"Lock::Appointment::" + toBase64(appointmentId),
	)
}

