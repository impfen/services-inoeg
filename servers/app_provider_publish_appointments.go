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
	//"encoding/hex"
	"github.com/kiebitz-oss/services"
	"github.com/kiebitz-oss/services/crypto"
	"time"
)

func (c *Appointments) publishAppointments(
	context services.Context,
	params *services.PublishAppointmentsSignedParams,
) services.Response {

	resp, providerKey := c.isProvider(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})

	if resp != nil {
		return resp
	}

	if expired(params.Data.Timestamp) {
		return context.Error(410, "signature expired", nil)
	}

	if len(params.Data.Appointments) > 500 {
		return context.Error(
			429,
			"max number of appointments per post exceeded",
			nil,
		)
	}

	pkd, err := providerKey.ProviderKeyData()

	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	providerId := crypto.Hash(pkd.Signing)
	// TODO: fix statistics generation
	//var bookedSlots, openSlots int64

	for _, appointment := range params.Data.Appointments {
		res := updateOrCreateAppointment(c, context, providerId, appointment)
		if res != nil { return res }
	}

	// TODO: fix statistics generation
	/*
	if c.meter != nil {

		now := time.Now().UTC().UnixNano()
		hexUID := hex.EncodeToString(providerId)

		addTokenStats := func(tw services.TimeWindow, data map[string]string) error {
			// we add the maximum of the open appointments
			if err := c.meter.AddMax("queues", "open", hexUID, data, tw, openSlots); err != nil {
				return err
			}
			// we add the maximum of the booked appointments
			if err := c.meter.AddMax("queues", "booked", hexUID, data, tw, bookedSlots); err != nil {
				return err
			}
			// we add the info that this provider is active
			if err := c.meter.AddOnce("queues", "active", hexUID, data, tw, 1); err != nil {
				return err
			}
			return nil
		}

		for _, twt := range tws {

			// generate the time window
			tw := twt(now)

			// global statistics
			if err := addTokenStats(tw, map[string]string{}); err != nil {
				services.Log.Error(err)
			}

			// statistics by zip code
			if err := addTokenStats(tw, map[string]string{
				"zipCode": pkd.QueueData.ZipCode,
			}); err != nil {
				services.Log.Error(err)
			}

		}

	}
	*/

	return context.Acknowledge()
}

func updateOrCreateAppointment (
	c *Appointments,
	context services.Context,
	providerId []byte,
	appointment *services.SignedAppointment,
) services.Response {

	// appointments are stored in a provider-specific key
	appointmentDatesByID := c.backend.AppointmentDatesByID(providerId)
	usedTokens := c.backend.UsedTokens()

	lock, err := c.LockAppointment(appointment.Data.ID)
	if err != nil {
		services.Log.Error(err)
		return LockError(context)
	}
	defer lock.Release()

	// check if there's an existing appointment
	if date, err := appointmentDatesByID.Get(appointment.Data.ID); err == nil {

		// delete old dates index
		if err := appointmentDatesByID.Del(appointment.Data.ID); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		appointmentsByDate :=
			c.backend.AppointmentsByDate(providerId, string(date))

		if existingAppointment, err := appointmentsByDate.Get(
			c.settings.Validate,
			appointment.Data.ID,
		); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		} else {

			// delete old properties indexes
			for k, v := range appointment.Data.Properties {
				appointmentDatesByProperty :=
					c.backend.AppointmentDatesByProperty(providerId, k, v)
				if err := appointmentDatesByProperty.Del(appointment.Data.ID); err != nil {
					services.Log.Error(err)
					return context.InternalError()
				}
			}

			// delete old appointment
			if err := appointmentsByDate.Del(appointment.Data.ID); err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}

			// deal with bookings
			bookings := make([]*services.Booking, 0)
			for _, existingSlotData := range existingAppointment.Data.SlotData {
				found := false
				for _, slotData := range appointment.Data.SlotData {
					if bytes.Equal(slotData.ID, existingSlotData.ID) {
						found = true
						break
					}
				}
				if found {
					// this slot has been preserved, if there's any booking for it we migrate it
					for _, booking := range existingAppointment.Bookings {
						if bytes.Equal(booking.ID, existingSlotData.ID) {
							bookings = append(bookings, booking)
							break
						}
					}
				} else {
					// this slot has been deleted, if there's any booking for it we delete it
					for _, booking := range existingAppointment.Bookings {
						if bytes.Equal(booking.ID, existingSlotData.ID) {
							// we re-enable the associated token
							if err := usedTokens.Del(booking.Token); err != nil {
								services.Log.Error(err)
								return context.InternalError()
							}
							break
						}
					}
				}
			}
			appointment.Bookings = bookings

		}
	}

	appointment.UpdatedAt = time.Now()

	date := appointment.Data.Timestamp.UTC().Format("2006-01-02")

	// create appointment
	appointmentsByDate := c.backend.AppointmentsByDate(providerId, date)
	if err := appointmentsByDate.Set(appointment); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	//create ByDate index
	if err := appointmentDatesByID.Set(appointment.Data.ID, date); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	// create ByProperty indexes
	for k, v := range appointment.Data.Properties {
		appointmentDatesByProperty := c.backend.AppointmentDatesByProperty(providerId, k, v)
		if err := appointmentDatesByProperty.Set(appointment.Data.ID, date); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	return nil
}
