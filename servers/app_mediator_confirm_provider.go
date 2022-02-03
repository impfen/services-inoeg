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
	"github.com/impfen/services-inoeg/databases"
)

// { id, key, providerData, keyData }, keyPair
func (c *Appointments) confirmProvider(
	context services.Context,
	params *services.ConfirmProviderSignedParams,
) services.Response {

	resp, _ := c.isMediator(context, &services.SignedParams{
		JSON:      params.JSON,
		Signature: params.Signature,
		PublicKey: params.PublicKey,
		Timestamp: params.Data.Timestamp,
	})

	if resp != nil {
		return resp
	}

	providerId := crypto.Hash(params.Data.SignedKeyData.Data.Signing)

	lock, err := c.LockProvider(providerId)
	if err != nil {
		services.Log.Error(err)
		return LockError(context)
	}
	defer lock.Release()

	keys := c.backend.Keys("providers")

	providerKey := &services.ActorKey{
		Data:      params.Data.SignedKeyData.JSON,
		Signature: params.Data.SignedKeyData.Signature,
		PublicKey: params.Data.SignedKeyData.PublicKey,
	}

	if err := keys.Set(providerId, providerKey); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	unverifiedProviderData := c.backend.UnverifiedProviderData()
	verifiedProviderData := c.backend.VerifiedProviderData()
	confirmedProviderData := c.backend.ConfirmedProviderData()
	publicProviderData := c.backend.PublicProviderData()

	oldPd, err := unverifiedProviderData.Get(providerId)

	if err != nil {
		if err == databases.NotFound {
			// maybe this provider has already been verified before...
			if oldPd, err = verifiedProviderData.Get(providerId); err != nil {
				if err == databases.NotFound {
					return context.NotFound()
				} else {
					services.Log.Error(err)
					return context.InternalError()
				}
			}
		} else {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	if err := unverifiedProviderData.Del(providerId); err != nil {
		if err != databases.NotFound {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	if err := verifiedProviderData.Set(providerId, oldPd); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	// we store a copy of the encrypted data for the provider to check
	if err := confirmedProviderData.Set(
		providerId,
		params.Data.ConfirmedProviderData,
	); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	if params.Data.PublicProviderData != nil {
		if err := publicProviderData.Set(
			providerId,
			params.Data.PublicProviderData,
		); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	return context.Acknowledge()
}
