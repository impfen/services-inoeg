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
	"bytes"
	"github.com/kiebitz-oss/services"
	"github.com/kiebitz-oss/services/databases"
	"time"
)

func (c *Appointments) isActiveProvider(
	context services.Context,
	id []byte,
) services.Response {

	if _, err := c.backend.Keys("providers").Get(id); err != nil {
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

	var result interface{}

	usedTokens := c.backend.UsedTokens()
	token := params.Data.SignedTokenData.Data.Token

	if ok, err := usedTokens.Has(token); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	} else if ok {
		return context.Error(401, "not authorized", nil)
	}

	// test if provider of the appointment is still active
	if res := c.isActiveProvider(context, params.Data.ProviderID); res != nil {
		return res
	}

	appointmentDatesByID := c.backend.AppointmentDatesByID(
		params.Data.ProviderID,
	)

	if date, err := appointmentDatesByID.Get(params.Data.ID); err != nil {
		services.Log.Errorf("Cannot get appointment by ID: %v", err)
		return context.InternalError()
	} else {

		appointmentsByDate := c.backend.AppointmentsByDate(
			params.Data.ProviderID,
			date,
		)

		// lock the appointment before attempting to retreive it
		lock, err := c.LockAppointment(params.Data.ID)
		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
		defer lock.Release()

		if signedAppointment, err := appointmentsByDate.Get(
			c.settings.Validate,
			params.Data.ID,
		); err != nil {

			services.Log.Errorf("Cannot get appointment by date: %v", err)
			return context.InternalError()

		} else {
			// we try to find an open slot
			foundSlot := false
			for _, slotData := range signedAppointment.Data.SlotData {

				found := false

				for _, booking := range signedAppointment.Bookings {
					if bytes.Equal(booking.ID, slotData.ID) {
						found = true
						break
					}
				}

				if found {
					continue
				}

				// this slot is open, we book it!

				booking := &services.Booking{
					PublicKey:     params.PublicKey,
					ID:            slotData.ID,
					Token:         token,
					EncryptedData: params.Data.EncryptedData,
				}

				signedAppointment.Bookings = append(signedAppointment.Bookings, booking)
				foundSlot = true

				result = booking

				break
			}

			if !foundSlot {
				return context.NotFound()
			}

			// we mark the token as used
			if err := usedTokens.Add(token); err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}

			signedAppointment.UpdatedAt = time.Now()

			if err := appointmentsByDate.Set(signedAppointment); err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}

		}

	}

	// TODO fix statistics
	/*
	if c.meter != nil {

		now := time.Now().UTC().UnixNano()

		for _, twt := range tws {

			// generate the time window
			tw := twt(now)

			// we add the info that a booking was made
			if err := c.meter.Add("queues", "bookings", map[string]string{}, tw, 1); err != nil {
				services.Log.Error(err)
			}

		}

	}
	*/

	return context.Result(result)

}
