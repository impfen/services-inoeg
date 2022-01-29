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
	"sort"
)

func (c *Appointments) getProviderAppointmentsByProperty(
	context services.Context,
	params *services.GetProviderAppointmentsByPropertySignedParams,
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

	pkd, err := providerKey.ProviderKeyData()

	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	providerId := crypto.Hash(pkd.Signing)

	// appointments are stored in a provider-specific key
	appointmentDatesByProperty := c.backend.AppointmentDatesByProperty(
		providerId,
		params.Data.Key,
		params.Data.Value,
	)
	allDates, err := appointmentDatesByProperty.GetAll()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	signedAppointments := make([]*services.SignedAppointment, 0)

	for appointmentId, dateStr := range allDates {

		if err != nil {
			services.Log.Error(err)
			continue
		}

		appointmentsByDate :=
			c.backend.AppointmentsByDate(providerId, string(dateStr))

		appointment, err := appointmentsByDate.Get(
			c.settings.Validate,
			[]byte(appointmentId),
		)
		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		signedAppointments = append(signedAppointments, appointment)
	}

	sort.Slice(signedAppointments, func (a, b int) bool {
		return signedAppointments[a].Data.Timestamp.Before(
			signedAppointments[b].Data.Timestamp,
		)
	})

	// public provider data structure
	publicProviderData := c.backend.PublicProviderData()
	providerData, err := publicProviderData.Get(providerId)
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}
	providerData.ID = providerId

	providerAppointments := &services.ProviderAppointments{
		Provider:     providerData,
		Appointments: signedAppointments,
	}

	return context.Result(providerAppointments)
}

