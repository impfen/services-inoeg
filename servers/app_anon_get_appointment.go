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

func (c *Appointments) getAppointment(
	context services.Context,
	params *services.GetAppointmentParams,
) services.Response {

	signedAppointment, err := c.backend.getAppointment(
		params.ID,
		params.ProviderID,
	)
	if err != nil {
		if err == databases.NotFound {
			return context.NotFound()
		}
		services.Log.Error(err)
		return context.InternalError()
	}

	// remove booking information
	for _, slot := range signedAppointment.Bookings {
		signedAppointment.BookedSlots = append(
			signedAppointment.BookedSlots,
			&services.Slot{ID: slot.ID},
		)
	}
	signedAppointment.Bookings = nil

	providerKey, err := c.backend.getProviderKeyByID(params.ProviderID)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	providerData, err := c.backend.getProviderByID(params.ProviderID)
	if err != nil {
		if err == databases.NotFound {
			return context.NotFound()
		}
		services.Log.Error(err)
		return context.InternalError()
	}

	mediatorKey, err := c.backend.findMediatorKey(providerKey.PublicKey)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	keyChain := &services.KeyChain{
		Provider: providerKey,
		Mediator: mediatorKey,
	}

	return context.Result(&services.ProviderAppointments{
		Provider:     providerData.PublicData,
		Appointments: []*services.SignedAppointment{signedAppointment},
		KeyChain:     keyChain,
	})

}
