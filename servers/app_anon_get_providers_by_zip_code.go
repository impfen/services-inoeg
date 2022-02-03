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
	"github.com/kiebitz-oss/services/crypto"
	"github.com/kiebitz-oss/services/databases"
	"sort"
)

func (c *Appointments) getProvidersByZipCode(
	context services.Context,
	params *services.GetProvidersByZipCodeParams,
) services.Response {

	// get all provider keys
	providerKeys, err := c.backend.getProviderKeys()
	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	// public provider data structure
	publicProviderData := c.backend.PublicProviderData()

	providers := []*services.SignedProviderData{}

	for _, providerKey := range providerKeys {

		if int64(len(providers)) >= c.settings.ResponseMaxProvider {
			break
		}

		pkd, err := providerKey.ProviderKeyData()
		if pkd.QueueData.ZipCode < params.ZipFrom ||
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
			}
			services.Log.Warning("provider data not found")
			continue
		}

		// we add the hash for convenience
		providerData.ID = providerID

		providers = append(providers, providerData)
	}

	sort.Slice(providers, func (a, b int) bool {
		return bytes.Compare(providers[a].ID, providers[b].ID) > 0
	})

	return context.Result(providers)

}

