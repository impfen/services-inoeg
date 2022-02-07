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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/kiebitz-oss/services"
	"github.com/kiebitz-oss/services/crypto"
)

/* Generates an HMAC based priority token and associated data structure. As the
token already gets signed with the token key it's currently a bit pointless to
use HMAC-based signature as the token. On the other hand this makes the tokens
deterministic, which can be useful to synchronize them in a decentralized setup
where different backends generate tokens and sign them with indidivual private
keys but still want to keep the priority tokens deterministic. Hence, we leave
this mechanism as is.
*/

var maxTokensError = fmt.Errorf("max tokens exceeded")

func (c *Appointments) priorityToken(
	userID []byte,
) (*services.PriorityToken, string, []byte, error) {

	// check that we are alowed to get another token
	if n, err := c.backend.userTokenAdd(userID, 1); err != nil {
		return nil, "", nil, err
	} else {
		if n > c.settings.MaxTokensPerUser {
			c.backend.userTokenAdd(userID, -1)
			return nil, "", nil, maxTokensError
		}
	}

	if n, err := c.backend.primaryTokenAdd(1); err != nil {
		return nil, "", nil, err
	} else {

		priorityToken := &services.PriorityToken{
			N: n,
		}

		if tokenData, err := priorityToken.Marshal(); err != nil {
			return nil, "", nil, err
		} else {
			h := hmac.New(sha256.New, c.settings.Secret)
			h.Write(tokenData)
			token := h.Sum(nil)
			return priorityToken, string(tokenData), token[:], nil
		}
	}
}

//{hash, code, publicKey}
// get a token for a given queue
func (c *Appointments) getToken(
	context services.Context,
	params *services.GetTokenParams,
) services.Response {

	tokenKey := c.settings.Key("token")
	if tokenKey == nil {
		services.Log.Error("token key missing")
		return context.InternalError()
	}

	var signedData *crypto.SignedStringData

	/*
	// TODO implement user codes
	codes := c.backend.Codes("user")

	if c.settings.UserCodesEnabled {
		notAuthorized := context.Error(401, "not authorized", nil)
		if params.Code == nil {
			return notAuthorized
		}
		if ok, err := codes.Has(params.Code); err != nil {
			services.Log.Error()
			return context.InternalError()
		} else if !ok {
			return notAuthorized
		}
	}
	*/

	userID := params.PublicKey

	if data, jsonData, token, err := c.priorityToken(userID); err != nil {
		services.Log.Error(err)
		if err == maxTokensError {
			context.Error(401, "maximum number of tokens exceeded", nil)
		} else {
			return context.InternalError()
		}
	} else {
		tokenData := &services.TokenData{
			Hash:      params.Hash,
			Token:     token,
			Data:      data,
			JSON:      jsonData,
			PublicKey: params.PublicKey,
		}

		td, err := json.Marshal(tokenData)

		if err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}

		if signedData, err = tokenKey.SignString(string(td)); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
	}

	/*
	// TODO implement user codes
	// if this is a new token we delete the user code
	if c.settings.UserCodesEnabled {
		score, err := codes.Score(params.Code)
		if err != nil && err != databases.NotFound {
			services.Log.Error(err)
			return context.InternalError()
		}

		score += 1

		if score > c.settings.UserCodesReuseLimit {
			if err := codes.Del(params.Code); err != nil {
				services.Log.Error(err)
				return context.InternalError()
			}
		} else if err := codes.AddToScore(params.Code, score); err != nil {
			services.Log.Error(err)
			return context.InternalError()
		}
	}
	*/

	return context.Result(signedData)

}
