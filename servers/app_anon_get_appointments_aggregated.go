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
	"time"
)

func (c *Appointments) getAppointmentsAggregated(context services.Context, params *services.GetAppointmentsByZipCodeParams) services.Response {

	// get all provider keys
	keys, err := c.getActorKeys()

	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	// get all neighboring zip codes for the given zip code
	neighbors := c.backend.Neighbors("zipCode", params.ZipCode)
	// public provider data structure
	publicProviderData := c.backend.PublicProviderData()

	allNeighbors, err := neighbors.Range(0, -1)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	distances := map[string]int64{}

	for _, neighbor := range allNeighbors {
		distances[string(neighbor.Data)] = neighbor.Score
	}

	providerAppointmentsList := []*services.AggregatedProviderAppointments{}

	for _, providerKey := range keys.Providers {

		if int64(len(providerAppointmentsList)) >= c.settings.AggregatedMaxProvider {
			break
		}

		pkd, err := providerKey.ProviderKeyData()

		if err != nil {
			services.Log.Error(err)
			continue
		}

		if pkd.QueueData.ZipCode != params.ZipCode {
			// check the distance of the zip codes don't match
			if distance, ok := distances[pkd.QueueData.ZipCode]; !ok {
				continue
			} else if distance > params.Radius {
				continue
			}
		}

		// the provider "ID" is the hash of the signing key
		hash := crypto.Hash(pkd.Signing)

		// fetch the full public data of the provider
		providerData, err := publicProviderData.Get(hash)

		if err != nil {
			if err != databases.NotFound {
				services.Log.Error(err)
			}
			services.Log.Warning("provider data not found")
			continue
		}

		// appointments are stored in a provider-specific key
		appointmentDatesByID := c.backend.AppointmentDatesByID(hash)
		// complexity: O(n) where n is the number of appointments of the provider
		allDates, err := appointmentDatesByID.GetAll()

		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		appointments := make([]*services.AppointmentAggregated, 0)

		visitedDates := make(map[string]bool)

	getAppointments:
		for _, dateStr := range allDates {

			if _, ok := visitedDates[string(dateStr)]; ok {
				continue
			} else {
				visitedDates[string(dateStr)] = true
			}

			date, err := time.Parse("2006-01-02", string(dateStr))
			if err != nil {
				services.Log.Error(err)
				continue
			}

			if date.Before(params.From) || date.After(params.To) {
				continue
			}

			appointmentsByDate := c.backend.AppointmentsByDate(hash, string(dateStr))
			allAppointments, err := appointmentsByDate.GetAll(c.settings.Validate)

			if err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}

			for _, signedAppointment := range allAppointments {

				slotN := len(signedAppointment.Data.SlotData) - len(signedAppointment.Bookings)

				// if all slots are booked we do not return the appointment
				if slotN <= 1 {
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

				if int64(len(appointments)) >= c.settings.AggregatedMaxAppointment {
					break getAppointments
				}
			}
		}

		if len(appointments) == 0 {
			continue
		}

		// we add the hash for convenience
		providerData.Data.ID = hash

		providerAppointments := &services.AggregatedProviderAppointments{
			Provider:     providerData.Data,
			Appointments: appointments,
		}

		providerAppointmentsList = append( providerAppointmentsList, providerAppointments)

	}

	return context.Result(providerAppointmentsList)
}

