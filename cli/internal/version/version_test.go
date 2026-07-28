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

package version

import "testing"

func TestEvaluateCompatibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cli        int
		minBackend int
		backend    int
		minCLI     int
		want       CompatibilityStatus
	}{
		{name: "compatible", cli: 2, minBackend: 1, backend: 2, minCLI: 1, want: CompatibilityCompatible},
		{name: "cli too old", cli: 1, minBackend: 1, backend: 2, minCLI: 2, want: CompatibilityCLITooOld},
		{name: "backend too old", cli: 2, minBackend: 2, backend: 1, minCLI: 1, want: CompatibilityBackendTooOld},
		{name: "explicit zero backend is pre-contract", cli: 1, minBackend: 1, backend: 0, minCLI: 0, want: CompatibilityBackendTooOld},
		{name: "both too old", cli: 1, minBackend: 2, backend: 1, minCLI: 2, want: CompatibilityBothTooOld},
		{name: "negative response is invalid", cli: 1, minBackend: 1, backend: -1, minCLI: 1, want: CompatibilityUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := evaluateCompatibility(test.cli, test.minBackend, test.backend, test.minCLI); got != test.want {
				t.Fatalf("evaluateCompatibility(%d, %d, %d, %d) = %q, want %q", test.cli, test.minBackend, test.backend, test.minCLI, got, test.want)
			}
		})
	}
}

func TestUserAgentUsesProductVersion(t *testing.T) {
	original := ProductVersion
	ProductVersion = "v0.4.0"
	t.Cleanup(func() { ProductVersion = original })

	if got := UserAgent(); got != "crater-cli/0.4.0" {
		t.Fatalf("UserAgent() = %q, want crater-cli/0.4.0", got)
	}
}
