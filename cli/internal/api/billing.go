package api

import (
	"net/url"
	"strconv"
	"strings"
)

type JobBillingListOptions struct {
	Admin    bool
	All      bool
	Username string
	Days     int
}

type JobBilling struct {
	JobName           string  `json:"jobName"`
	Name              string  `json:"name"`
	BilledPointsTotal float64 `json:"billedPointsTotal"`
}

func (c *Client) ListJobBilling(options JobBillingListOptions) ([]JobBilling, error) {
	path := VCJobBillingPath
	includeDays := false
	username := strings.TrimSpace(options.Username)

	switch {
	case options.Admin && username != "":
		path = AdminVCJobBillingPath + "/user/" + url.PathEscape(username)
		includeDays = true
	case options.Admin:
		path = AdminVCJobBillingPath
		includeDays = true
	case username != "":
		path = VCJobBillingPath + "/user/" + url.PathEscape(username)
		includeDays = true
	case options.All:
		path = VCJobBillingPath + "/all"
		includeDays = true
	}

	var result Response[[]JobBilling]
	request := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result)
	if includeDays {
		request.SetQueryParam("days", strconv.Itoa(options.Days))
	}
	resp, err := request.Get(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	if result.Data == nil {
		result.Data = []JobBilling{}
	}
	return result.Data, nil
}
