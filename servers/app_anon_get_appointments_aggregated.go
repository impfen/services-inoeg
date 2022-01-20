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
	"github.com/kiebitz-oss/services/crypto"
	"github.com/kiebitz-oss/services/databases"
	"sort"
	"time"
)

func (c *Appointments) getAppointmentsAggregated(
	context services.Context,
	params *services.GetAppointmentsAggregatedParams,
) services.Response {

	// get all provider keys
	keys, err := c.getActorKeys()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	// get the current time
	now := time.Now()

	// public provider data structure
	publicProviderData := c.backend.PublicProviderData()

	providerAppointmentsList := []*services.AggregatedProviderAppointments{}

	for _, providerKey := range keys.Providers {

		pkd, err := providerKey.ProviderKeyData()
		if err != nil {
			services.Log.Error(err)
			continue
		}

		if
			pkd.QueueData.ZipCode < params.ZipFrom ||
			pkd.QueueData.ZipCode > params.ZipTo {
			continue
		}

		// the provider "ID" is the hash of the signing key
		providerID := crypto.Hash(pkd.Signing)

		// fetch the full public data of the provider
		providerData, err := publicProviderData.Get(providerID)

		if err != nil {
			if err != databases.NotFound {
				services.Log.Error(err)
			} else {
				services.Log.Warning("provider data not found")
			}
			continue
		}

		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		appointments := make([]*services.AppointmentAggregated, 0)

		for day := 0; day <= 7; day++ {

			appointmentsByDate := c.backend.AppointmentsByDate(
				providerID,
				params.Date.AddDate(0, 0, day).Format("2006-01-02"),
			)
			allAppointments, err := appointmentsByDate.GetAll(c.settings.Validate)

			if err != nil {
				if err == databases.NotFound {
					continue
				} else {
					services.Log.Error(err)
					return context.InternalError()
				}
			}

			for _, signedAppointment := range allAppointments {

				slotN :=
					len(signedAppointment.Data.SlotData) - len(signedAppointment.Bookings)

				// if all slots are booked or the appointment is in the past, we do not
				// return it
				if slotN < 1 || signedAppointment.Data.Timestamp.Before(now) {
					continue
				}

				appointment := &services.AppointmentAggregated{
					ID:         signedAppointment.Data.ID,
					Duration:   signedAppointment.Data.Duration,
					Properties: signedAppointment.Data.Properties,
					SlotN:      slotN,
					Timestamp:  signedAppointment.Data.Timestamp,
				}

				appointments = append(appointments, appointment)

			}

			if len(appointments) > 25 { break }
		}

		sort.Slice(appointments, func (a, b int) bool {
			return appointments[a].Timestamp.Before(appointments[b].Timestamp)
		})

		// we add the provider id for convenience
		providerData.Data.ID = providerID

		providerAppointments := &services.AggregatedProviderAppointments{
			Provider:     providerData.Data,
			Appointments: appointments,
		}

		providerAppointmentsList =
			append( providerAppointmentsList, providerAppointments)

	}

	return context.Result(providerAppointmentsList)
}

