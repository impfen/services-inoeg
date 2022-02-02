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
	"encoding/base64"
)

type Lock interface {
	Release() error
}

func toBase64 (bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// appointment locks prevent concurrent changes in the appointment data (like
// bookings) which may lead to race conditions resulting in bookings getting
// lost
func (c *Appointments) LockAppointment (
	appointmentId []byte,
) (Lock, error) {
	return nil, nil
}

// provider locks prohibit the provider and mediator to change data concurrently
// which may lead to inconsisten data
func (c *Appointments) LockProvider (
	providerId []byte,
) (Lock, error) {
	return nil, nil
}

// token locks prevent double spending of tokens
func (c *Appointments) LockToken (
	token []byte,
) (Lock, error) {
	return nil, nil
}

// user lock prevents race conditions when checking the token limit per user
func (c *Appointments) LockUser (
	userId []byte,
) (Lock, error) {
	return nil, nil
}

func LockError (context services.Context) services.Response {
	return context.Error(503, "lock timeout", nil)
}
