// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"time"
)

const compatibilityCheckTimeout = 2 * time.Second

type CompatibilityInfo struct {
	APIVersion                int    `json:"apiVersion"`
	MinSupportedCLIAPIVersion int    `json:"minSupportedCliApiVersion"`
	AppVersion                string `json:"appVersion"`
	ShortCommitSHA            string `json:"shortCommitSHA"`
	BuildType                 string `json:"buildType"`
	BuildTime                 string `json:"buildTime"`
	VersionsReported          bool   `json:"-"`
}

type compatibilityPayload struct {
	APIVersion                *int   `json:"apiVersion"`
	MinSupportedCLIAPIVersion *int   `json:"minSupportedCliApiVersion"`
	AppVersion                string `json:"appVersion"`
	ShortCommitSHA            string `json:"shortCommitSHA"`
	BuildType                 string `json:"buildType"`
	BuildTime                 string `json:"buildTime"`
}

func (c *Client) GetCompatibility() (*CompatibilityInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityCheckTimeout)
	defer cancel()

	var result Response[compatibilityPayload]
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(CompatibilityPath)
	// req/v3 may return both an HTTP response and an unmarshalling error when an
	// older server responds with a plain-text error body. Preserve the HTTP
	// semantics before classifying errors that happened without a response.
	if resp != nil && resp.GetStatusCode() != 0 {
		if responseErr := errorFromResponse(resp, result.Code, result.Message); responseErr != nil {
			return nil, responseErr
		}
	}
	if err != nil {
		if resp != nil && resp.GetStatusCode() != 0 {
			return nil, fmt.Errorf("decode compatibility response: %w", err)
		}
		return nil, &NetworkError{Cause: err}
	}
	info := &CompatibilityInfo{}
	if result.Data.APIVersion != nil {
		info.APIVersion = *result.Data.APIVersion
	}
	if result.Data.MinSupportedCLIAPIVersion != nil {
		info.MinSupportedCLIAPIVersion = *result.Data.MinSupportedCLIAPIVersion
	}
	info.AppVersion = result.Data.AppVersion
	info.ShortCommitSHA = result.Data.ShortCommitSHA
	info.BuildType = result.Data.BuildType
	info.BuildTime = result.Data.BuildTime
	info.VersionsReported = result.Data.APIVersion != nil && result.Data.MinSupportedCLIAPIVersion != nil
	return info, nil
}
