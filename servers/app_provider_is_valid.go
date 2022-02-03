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
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/crypto"
)

func (c *Appointments) isValidProvider(
	context services.Context,
	params *services.CheckProviderDataSignedParams,
) services.Response {

	providerId := crypto.Hash(params.PublicKey)

	lock, err := c.LockProvider(providerId)
	if err != nil {
		services.Log.Error(err)
		return LockError(context)
	}
	defer lock.Release()

	validatedErr, _ := c.isProvider(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})

	pendingErr := c.isPendingProvider(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})

	if validatedErr != nil && pendingErr != nil{

		if context.IsInternalError(validatedErr) {
			return validatedErr

		} else if context.IsInternalError(pendingErr) {
			return pendingErr

		} else {
			return context.Result(false)

		}
	}

	return context.Result(true)
}

