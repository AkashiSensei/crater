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

package compatibility_test

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/snaptest"
)

const goldenStemCompatibility = "compatibility"

type compatibilityCase struct {
	snapshot     snaptest.Case
	httpMode     string
	emptySession bool
}

func TestCompatibilitySnapshotsEN(t *testing.T) {
	runCompatibilitySnapshots(t, "en")
}

func TestCompatibilitySnapshotsZhCN(t *testing.T) {
	runCompatibilitySnapshots(t, "zh-CN")
}

func runCompatibilitySnapshots(t *testing.T, lang string) {
	t.Helper()
	testCases := []compatibilityCase{
		{
			snapshot: snaptest.Case{ID: "01-compatible-active-platform-nojson", Args: []string{"compatibility", "--no-interactive"}},
			httpMode: "compatibility-compatible",
		},
		{
			snapshot: snaptest.Case{ID: "02-compatible-json", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive", "--json"}},
			httpMode: "compatibility-compatible",
		},
		{
			snapshot: snaptest.Case{ID: "03-incompatible-nojson", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive"}},
			httpMode: "compatibility-cli-too-old",
		},
		{
			snapshot: snaptest.Case{ID: "04-incompatible-json", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive", "--json"}},
			httpMode: "compatibility-cli-too-old",
		},
		{
			snapshot: snaptest.Case{ID: "05-endpoint-unavailable-nojson", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive"}},
			httpMode: "compatibility-unavailable",
		},
		{
			snapshot: snaptest.Case{ID: "06-endpoint-unavailable-json", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive", "--json"}},
			httpMode: "compatibility-unavailable",
		},
		{
			snapshot: snaptest.Case{ID: "07-backend-zero-nojson", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive"}},
			httpMode: "compatibility-backend-zero",
		},
		{
			snapshot: snaptest.Case{ID: "08-backend-zero-json", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive", "--json"}},
			httpMode: "compatibility-backend-zero",
		},
		{
			snapshot:     snaptest.Case{ID: "09-no-active-platform-nojson", Args: []string{"compatibility", "--no-interactive"}},
			httpMode:     "compatibility-compatible",
			emptySession: true,
		},
		{
			snapshot:     snaptest.Case{ID: "10-no-active-platform-json", Args: []string{"compatibility", "--no-interactive", "--json"}},
			httpMode:     "compatibility-compatible",
			emptySession: true,
		},
		{
			snapshot: snaptest.Case{ID: "11-network-timeout-nojson", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive"}},
			httpMode: "timeout",
		},
		{
			snapshot: snaptest.Case{ID: "12-network-timeout-json", Args: []string{"compatibility", "--platform", "https://example.invalid", "--no-interactive", "--json"}},
			httpMode: "timeout",
		},
	}

	bin := snaptest.CraterExecutable(t)
	baseEnv := snaptest.EnvMinimal(t.TempDir(), lang)
	cases := make([]snaptest.Case, len(testCases))
	results := make([]*snaptest.Result, len(testCases))
	for i := range testCases {
		cases[i] = testCases[i].snapshot
		env := append([]string{}, baseEnv...)
		env = append(env, "CRATER_TEST_SANDBOX_HTTP="+testCases[i].httpMode)
		if testCases[i].emptySession {
			env = append(env, "CRATER_TEST_SANDBOX_SESSION_EMPTY=1")
		}
		result, err := snaptest.Run(bin, env, cases[i].Args)
		if err != nil {
			t.Fatalf("case %s: %v", cases[i].ID, err)
		}
		results[i] = result
	}

	path := snaptest.GoldenFileT(t, "compatibility", goldenStemCompatibility, lang)
	update := os.Getenv("UPDATE_SNAPSHOTS") == "1" || os.Getenv("UPDATE_SNAPSHOTS") == "true"
	if err := snaptest.MatchOrUpdateGolden(path, lang, cases, results, update); err != nil {
		t.Fatal(err)
	}
}
