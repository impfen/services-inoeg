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
	"github.com/impfen/services-inoeg"
	"sort"
)

// mediator-only endpoint
// { limit }, keyPair
func (c *Appointments) getProviders(
	context services.Context,
	params *services.GetProvidersDataSignedParams,
) services.Response {

	resp, _ := c.isMediator(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})
	if resp != nil { return resp }

	providerStatus := c.backend.ProviderStatus()
	pdEntries := []*services.RawProviderData{}

	verifiedProviderData := c.backend.VerifiedProviderData()
	verifiedProviderDataMap, err := verifiedProviderData.GetAll()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	for pId, pd := range verifiedProviderDataMap {
		pd.ID = []byte(pId)
		pd.Verified = true
		if status, err := providerStatus.Get([]byte(pId)); err != nil {
			services.Log.Errorf("provider %s has no status %#v: ", pId, err)
			pd.Status = "UNKNOWN"
		} else {
			pd.Status = status
		}
		pdEntries = append(pdEntries, pd)
	}

	unverifiedProviderData := c.backend.UnverifiedProviderData()
	unverifiedproviderDataMap, err := unverifiedProviderData.GetAll()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	for pId, pd := range unverifiedproviderDataMap {
		pd.ID = []byte(pId)
		pd.Verified = false
		if status, err := providerStatus.Get([]byte(pId)); err != nil {
			services.Log.Errorf("provider %s has no status %#v: ", pId, err)
			pd.Status = "UNKNOWN"
		} else {
			pd.Status = status
		}
		pdEntries = append(pdEntries, pd)
	}

	sort.Slice(pdEntries, func (a, b int) bool {
		return bytes.Compare(pdEntries[a].ID, pdEntries[b].ID) > 0
	})

	return context.Result(pdEntries)

}
