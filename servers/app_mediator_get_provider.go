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

// mediator-only endpoint
func (c *Appointments) getProviderData(
	context services.Context,
	params *services.GetProviderDataSignedParams,
) services.Response {

	resp, _ := c.isMediator(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})
	if resp != nil { return resp }

	verifiedProviderData := c.backend.VerifiedProviderData()
	verPro, verProErr := verifiedProviderData.Get(params.Data.ProviderID)
	if verProErr != nil && verProErr != databases.NotFound {
		services.Log.Error(verProErr)
		return context.InternalError()
	}

	unverifiedProviderData := c.backend.UnverifiedProviderData()
	unPro, unProErr := unverifiedProviderData.Get(params.Data.ProviderID)
	if unProErr != nil && unProErr != databases.NotFound {
		services.Log.Error(unProErr)
		return context.InternalError()
	}

	if verPro == nil && unPro == nil {
		return context.NotFound()
	}

	if unPro != nil {
		unPro.ID       = params.Data.ProviderID
		unPro.Verified = false
	}

	if verPro != nil {
		verPro.ID       = params.Data.ProviderID
		verPro.Verified = true
		if unPro == nil {
			unPro = verPro
		}
	}

	return context.Result( &services.GetProviderResult{
		UnverifiedData: unPro,
		VerifiedData:   verPro,
	})

}
