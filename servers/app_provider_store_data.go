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

// { id, encryptedData, code }, keyPair
func (c *Appointments) storeProviderData(
	context services.Context,
	params *services.StoreProviderDataSignedParams,
) services.Response {

	/* we verify the signature (without veryfing e.g. the provenance of the key)
	 this is important as we use the public key as an identifier for the provider
	 data so we need to make sure the caller is actually in possession of the key
	*/
	if ok, err := crypto.VerifyWithBytes(
		[]byte(params.JSON),
		params.Signature,
		params.PublicKey,
	); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	} else if !ok {
		return context.Error(400, "invalid signature", nil)
	}

	if expired(params.Data.Timestamp) {
		return context.Error(410, "signature expired", nil)
	}

	// TODO: add one-time use check

	providerID := crypto.Hash(params.PublicKey)

	verifiedProviderData := c.backend.VerifiedProviderData()
	providerData := c.backend.UnverifiedProviderData()
	providerStatus := c.backend.ProviderStatus()
	codes := c.backend.Codes("provider")

	existingData := false
	if result, err := verifiedProviderData.Get(providerID); err != nil {
		if err != databases.NotFound {
			services.Log.Error(err)
			return context.InternalError()
		}
	} else if result != nil {
		existingData = true
	}

	if (!existingData) && c.settings.ProviderCodesEnabled {
		if params.Data.Code == nil {
			return context.Error(401, "not authorized", nil)
		}
		if ok, err := codes.Has(params.Data.Code); err != nil {
			services.Log.Error()
			return context.InternalError()
		} else if !ok {
			return context.Error(401, "not authorized", nil)
		}
	}

	// aquire a lock before writing new data
	lock, err := c.LockProvider(providerID)
	if err != nil {
		services.Log.Error(err)
		return LockError(context)
	}
	defer lock.Release()

	rawProviderData := &services.RawProviderData{
		ID: providerID,
		EncryptedData: params.Data.EncryptedData,
	}

	if err := providerData.Set(providerID, rawProviderData); err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	if existingData {
		providerStatus.Set(providerID, "CHANGED")
	} else {
		providerStatus.Set(providerID, "UNVERIFIED")
	}

	// we delete the provider code
	if c.settings.ProviderCodesEnabled {
		score, err := codes.Score(params.Data.Code)
		if err != nil && err != databases.NotFound {
			services.Log.Error(err)
			return context.InternalError()
		}

		score += 1

		if score > c.settings.ProviderCodesReuseLimit {
			if err := codes.Del(params.Data.Code); err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}
		} else if err := codes.AddToScore(params.Data.Code, score); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	return context.Acknowledge()
}
