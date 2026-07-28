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
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	internalversion "github.com/raids-lab/crater/internal/version"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewCLICompatibilityMgr)
}

type CLICompatibilityMgr struct {
	name string
}

func NewCLICompatibilityMgr(_ *RegisterConfig) Manager {
	return &CLICompatibilityMgr{name: "cli"}
}

func (mgr *CLICompatibilityMgr) GetName() string                      { return mgr.name }
func (mgr *CLICompatibilityMgr) RegisterProtected(_ *gin.RouterGroup) {}
func (mgr *CLICompatibilityMgr) RegisterAdmin(_ *gin.RouterGroup)     {}

func (mgr *CLICompatibilityMgr) RegisterPublic(g *gin.RouterGroup) {
	g.GET("compatibility", mgr.GetCompatibility)
}

type CLICompatibilityInfo struct {
	APIVersion                int    `json:"apiVersion"`
	MinSupportedCLIAPIVersion int    `json:"minSupportedCliApiVersion"`
	AppVersion                string `json:"appVersion"`
	ShortCommitSHA            string `json:"shortCommitSHA"`
	BuildType                 string `json:"buildType"`
	BuildTime                 string `json:"buildTime"`
}

// GetCompatibility godoc
//
//	@Summary		获取 CLI API 兼容信息
//	@Description	返回后端当前 CLI API 版本、最低支持的 CLI API 版本及构建信息；构建信息不参与兼容性判断，该接口只提供诊断信息，不拦截业务请求
//	@Tags			CLI
//	@Produce		json
//	@Success		200	{object}	resputil.Response[CLICompatibilityInfo]	"CLI API 兼容信息"
//	@Router			/cli/compatibility [get]
func (mgr *CLICompatibilityMgr) GetCompatibility(c *gin.Context) {
	versionInfo := GetVersionInfo()
	resputil.Success(c, CLICompatibilityInfo{
		APIVersion:                internalversion.APIVersion,
		MinSupportedCLIAPIVersion: internalversion.MinSupportedCLIAPIVersion,
		AppVersion:                versionInfo.AppVersion,
		ShortCommitSHA:            shortCommitSHA(versionInfo.CommitSHA),
		BuildType:                 versionInfo.BuildType,
		BuildTime:                 versionInfo.BuildTime,
	})
}

func shortCommitSHA(commitSHA string) string {
	const shortCommitSHALength = 7

	commitSHA = strings.TrimSpace(commitSHA)
	if len(commitSHA) <= shortCommitSHALength {
		return commitSHA
	}
	return commitSHA[:shortCommitSHALength]
}
