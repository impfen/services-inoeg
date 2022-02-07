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
)

func (c *Appointments) getProviderAppointments(
	context services.Context,
	params *services.GetProviderAppointmentsSignedParams,
) services.Response {

	resp, _ := c.isProvider(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})
	if resp != nil { return resp }

	providerID := crypto.Hash(params.PublicKey)

	signedAppointments, err := c.backend.getAppointmentsByDate(
		providerID,
		params.Data.From,
		params.Data.To,
	)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	/*

	// appointments are stored in a provider-specific key
	appointmentDatesByID := c.backend.AppointmentDatesByID(providerId)
	allDates, err := appointmentDatesByID.GetAll()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	signedAppointments := make([]*services.SignedAppointment, 0)
	visitedDates := make(map[string]bool)

	dateFrom := params.Data.From.Format("2006-01-02")
	dateTo := params.Data.To.Format("2006-01-02")

	for _, date := range allDates {

		dateStr := string(date)

		if _, ok := visitedDates[dateStr]; ok {
			continue
		} else {
			visitedDates[dateStr] = true
		}

		if dateStr < dateFrom || dateStr > dateTo {continue}

		appointmentsByDate := c.backend.AppointmentsByDate(providerId, dateStr)

		allAppointments, err := appointmentsByDate.GetAll(c.settings.Validate)

		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		for _, appointment := range allAppointments {
			// if the updatedSince parameter is given we only return appointments that
			// have been updated since the given time
			isUpdated := params.Data.UpdatedSince != nil && (
				params.Data.UpdatedSince.After(appointment.UpdatedAt) ||
				params.Data.UpdatedSince.Equal(appointment.UpdatedAt) )
			if isUpdated {continue}

			signedAppointments = append(signedAppointments, appointment)
		}
	}

	sort.Slice(signedAppointments, func (a, b int) bool {
		return signedAppointments[a].Data.Timestamp.Before(
			signedAppointments[b].Data.Timestamp,
		)
	})

	*/

	// public provider data structure
	providerData, err := c.backend.getPublicProviderByID(providerID)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	providerAppointments := &services.ProviderAppointments{
		Provider:     providerData,
		Appointments: signedAppointments,
	}

	return context.Result(providerAppointments)
}
