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

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	internalversion "github.com/raids-lab/crater/internal/version"
)

func TestCLICompatibilityEndpointIsPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalVersionInfo := GetVersionInfo()
	t.Cleanup(func() {
		SetVersionInfo(
			originalVersionInfo.AppVersion,
			originalVersionInfo.CommitSHA,
			originalVersionInfo.BuildType,
			originalVersionInfo.BuildTime,
		)
	})
	SetVersionInfo(
		"v1.1.1",
		"f42b0c2a1234567890abcdef1234567890abcdef",
		"release",
		"2026-07-26T08:30:00Z",
	)

	mgr := NewCLICompatibilityMgr(nil)
	router := gin.New()
	mgr.RegisterPublic(router.Group("/api/cli"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/cli/compatibility", http.NoBody)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code int                  `json:"code"`
		Data CLICompatibilityInfo `json:"data"`
		Msg  string               `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 {
		t.Fatalf("code = %d, want 0", response.Code)
	}
	if response.Data.APIVersion != internalversion.APIVersion {
		t.Fatalf("apiVersion = %d, want %d", response.Data.APIVersion, internalversion.APIVersion)
	}
	if response.Data.MinSupportedCLIAPIVersion != internalversion.MinSupportedCLIAPIVersion {
		t.Fatalf("minSupportedCliApiVersion = %d, want %d", response.Data.MinSupportedCLIAPIVersion, internalversion.MinSupportedCLIAPIVersion)
	}
	if response.Data.AppVersion != "v1.1.1" {
		t.Fatalf("appVersion = %q, want %q", response.Data.AppVersion, "v1.1.1")
	}
	if response.Data.ShortCommitSHA != "f42b0c2" {
		t.Fatalf("shortCommitSHA = %q, want %q", response.Data.ShortCommitSHA, "f42b0c2")
	}
	if response.Data.BuildType != "release" {
		t.Fatalf("buildType = %q, want %q", response.Data.BuildType, "release")
	}
	if response.Data.BuildTime != "2026-07-26T08:30:00Z" {
		t.Fatalf("buildTime = %q, want %q", response.Data.BuildTime, "2026-07-26T08:30:00Z")
	}
}

func TestShortCommitSHA(t *testing.T) {
	tests := []struct {
		name      string
		commitSHA string
		want      string
	}{
		{name: "full SHA", commitSHA: "f42b0c2a1234567890abcdef1234567890abcdef", want: "f42b0c2"},
		{name: "already short", commitSHA: "f42b0c2", want: "f42b0c2"},
		{name: "unknown", commitSHA: "unknown", want: "unknown"},
		{name: "trim whitespace", commitSHA: "  f42b0c2a  ", want: "f42b0c2"},
		{name: "empty", commitSHA: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortCommitSHA(tt.commitSHA); got != tt.want {
				t.Fatalf("shortCommitSHA(%q) = %q, want %q", tt.commitSHA, got, tt.want)
			}
		})
	}
}
