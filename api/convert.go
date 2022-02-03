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

package api

import (
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/jsonrpc"
	"github.com/impfen/services-inoeg/rest"
	"github.com/kiprotect/go-helpers/forms"
)

var APIDocForm = &forms.Form{}

type APIDocParams struct {
}

func makeJSONRPCDoc(methods map[string]*jsonrpc.Method, api *API) interface{} {
	return func(context services.Context, params *APIDocParams) services.Response {
		return context.Result(api)
	}
}

func (c *API) ToJSONRPC(validateSettings *services.ValidateSettings) (jsonrpc.Handler, error) {
	methods := map[string]*jsonrpc.Method{}
	for _, endpoint := range c.Endpoints {
		methods[endpoint.Name] = &jsonrpc.Method{
			Form:    endpoint.Form,
			Handler: endpoint.Handler,
		}
	}
	methods["_doc"] = &jsonrpc.Method{
		Form:    APIDocForm,
		Handler: makeJSONRPCDoc(methods, c),
	}
	return jsonrpc.MethodsHandler(methods, validateSettings)
}

func (c *API) ToREST(validateSettings *services.ValidateSettings) (rest.Handler, error) {
	methods := map[string]*rest.Method{}
	for _, endpoint := range c.Endpoints {
		if endpoint.REST == nil {
			continue
		}
		methods[endpoint.Name] = &rest.Method{
			Path:    endpoint.REST.Path,
			Method:  string(endpoint.REST.Method),
			Form:    endpoint.Form,
			Handler: endpoint.Handler,
		}
	}
	return rest.MethodsHandler(methods, validateSettings)
}
