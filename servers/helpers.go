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
	"encoding/base64"
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/crypto"
	"time"
)

// in principle JSON will encode binary data as base64, but we do the conversion
// explicitly just to avoid any potential inconsistencies that might arise in the future...
func Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// in principle JSON will encode binary data as base64, but we do the conversion
// explicitly just to avoid any potential inconsistencies that might arise in the future...
func EncodeSlice(data [][]byte) []string {
	strings := make([]string, len(data))
	for i, d := range data {
		strings[i] = base64.StdEncoding.EncodeToString(d)
	}
	return strings
}

func findActorKey(keys []*services.ActorKey, publicKeyOrID []byte) (*services.ActorKey, error) {
	for _, key := range keys {
		if bytes.Equal(key.ID, publicKeyOrID) {
			return key, nil
		}
		if akd, err := key.KeyData(); err != nil {
			services.Log.Error(err)
			continue
		} else if bytes.Equal(akd.Signing, publicKeyOrID) {
			return key, nil
		}
	}
	return nil, nil
}

func isRoot(context services.Context, data, signature []byte, timestamp time.Time, keys []*crypto.Key) services.Response {
	rootKey := services.Key(keys, "root")
	if rootKey == nil {
		services.Log.Error("root key missing")
		return context.InternalError()
	}
	if ok, err := rootKey.Verify(&crypto.SignedData{
		Data:      data,
		Signature: signature,
	}); !ok {
		return context.Error(403, "invalid signature", nil)
	} else if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}
	if expired(timestamp) {
		return context.Error(410, "signature expired", nil)
	}
	return nil
}

func expired(timestamp time.Time) bool {
	return time.Now().Add(-time.Minute).After(timestamp)
}
