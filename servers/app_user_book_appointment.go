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
	"github.com/kiebitz-oss/services/databases"
)

func (c *Appointments) isActiveProvider(
	context services.Context,
	id []byte,
) services.Response {

	if _, err := c.backend.getProviderKeyByID(id); err != nil {
		if err == databases.NotFound {
			return context.Error(404, "provider not found", nil)
		}
	}

	return nil
}

func (c *Appointments) bookAppointment(
	context services.Context,
	params *services.BookAppointmentSignedParams,
) services.Response {

	if resp := c.isUser(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		ExtraData: params.Data.SignedTokenData,
		Timestamp: params.Data.Timestamp,
	}); resp != nil {
		return resp
	}

	token := params.Data.SignedTokenData.Data.Token

	slotID, err := c.backend.bookAppointment(
		params.Data.ProviderID,
		params.Data.ID, // appointment id
		params.PublicKey,
		token,
		params.Data.EncryptedData,
	)
	if err != nil {
		if err == databases.ErrTokenUsed {
			return context.Error(401, "not authorized", nil)
		} else if err == databases.NotFound {
			return context.Error(404, "not found", nil)
		} else {
			services.Log.Error(err)
			return LockError(context)
		}
	}

	booking := &services.Booking{
		PublicKey:     params.PublicKey,
		ID:            slotID,
		Token:         token,
		EncryptedData: params.Data.EncryptedData,
	}

	return context.Result(booking)
}
