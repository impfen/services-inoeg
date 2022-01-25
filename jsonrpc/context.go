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

package jsonrpc

import (
	"fmt"
	"github.com/kiebitz-oss/services"
	"regexp"
	"strconv"
	"strings"
)

var idRegexp = regexp.MustCompile(`^n:(-?\d{1,32})$`)
var idNRegexp = regexp.MustCompile(`^(n+):(-?\d{1,32})$`)

type Context struct {
	Request *Request
}

func convertID(id interface{}) interface{} {
	if strValue, ok := id.(string); ok {
		if matches := idRegexp.FindStringSubmatch(strValue); matches != nil {
			// we convert this value back to a number
			if n, err := strconv.ParseInt(matches[1], 10, 64); err != nil {
				// this should not happen, if it does we log the error and
				// return the string value (could only be an overflow)
				services.Log.Error(err)
				return id
			} else {
				return n
			}
		}
		if matches := idNRegexp.FindStringSubmatch(strValue); matches != nil && len(matches[1])%2 == 0 {
			return fmt.Sprintf("%s:%s", strings.Repeat("n", len(matches[1])/2), matches[2])
		}
	}
	// we do not convert anything
	return id
}

func (c *Context) Result(data interface{}) services.Response {

	return &Response{
		ID:      convertID(c.Request.ID),
		Result:  data,
		JSONRPC: "2.0",
	}
}

func (c *Context) Error(code int, message string, data interface{}) services.Response {
	return &Response{
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		JSONRPC: "2.0",
		ID:      convertID(c.Request.ID),
	}
}

func (c *Context) Params() map[string]interface{} {
	return c.Request.Params
}

func (c *Context) NotFound() services.Response {
	return c.Error(404, "not found", nil)
}

func (c *Context) Acknowledge() services.Response {
	return c.Result("ok")
}

func (c *Context) Nil() services.Response {
	return c.Result(nil)
}

func (c *Context) MethodNotFound() services.Response {
	return c.Error(-32601, "method not found", nil)
}

func (c *Context) InvalidParams(err error) services.Response {
	return c.Error(-32602, "invalid params", err)
}

func (c *Context) InternalError() services.Response {
	return c.Error(-32603, "internal error", nil)
}

func (c *Context) IsInternalError(resp services.Response) bool {
	if resp == nil {
		return false
	}
	intErr := c.InternalError().(*Response)
	a := resp.(*Response).Error.Code    == intErr.Error.Code
	b := resp.(*Response).Error.Message == intErr.Error.Message
	return a && b
}
