package banana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// APIError is returned when GameBanana responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Code       string                // e.g. "INPUT_ERRORS"
	Fields     map[string]FieldError // per-parameter errors, keyed by parameter name
}

// FieldError describes a problem with a single request parameter.
type FieldError struct {
	Code    string `json:"_sErrorCode"`
	Message string `json:"_sErrorMessage"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "banana: HTTP %d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " %s", e.Code)
	}

	names := make([]string, 0, len(e.Fields))
	for name := range e.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		sep := ", "
		if i == 0 {
			sep = ": "
		}
		fmt.Fprintf(&b, "%s%s: %s", sep, name, e.Fields[name].Message)
	}
	return b.String()
}

func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{StatusCode: resp.StatusCode}

	var body struct {
		Code   string                     `json:"_sErrorCode"`
		Fields map[string]json.RawMessage `json:"_aErrorData"`
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if json.Unmarshal(data, &body) != nil {
		return apiErr
	}

	apiErr.Code = body.Code
	for name, raw := range body.Fields {
		// _aErrorData values aren't guaranteed to be objects; skip ones that aren't.
		var fe FieldError
		if json.Unmarshal(raw, &fe) == nil {
			if apiErr.Fields == nil {
				apiErr.Fields = make(map[string]FieldError)
			}
			apiErr.Fields[name] = fe
		}
	}
	return apiErr
}
