/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const DEVELOPMENT_VERSION_PATTERN = /^(\d+\.\d+\.\d+)\+dev\.(\d+)\.g[0-9a-f]+$/i

type VersionValueProps = {
  version?: string
  buildType?: string
  fallback: string
}

function summarizeVersion(version: string, buildType?: string) {
  if (buildType !== 'development') {
    return version
  }

  const match = DEVELOPMENT_VERSION_PATTERN.exec(version)
  if (!match) {
    return version
  }

  return `${match[1]}+dev.${match[2]}`
}

export function VersionValue({ version, buildType, fallback }: VersionValueProps) {
  if (!version) {
    return <span className="text-foreground font-mono font-semibold">{fallback}</span>
  }

  const displayVersion = summarizeVersion(version, buildType)
  const hasHiddenMetadata = displayVersion !== version

  if (!hasHiddenMetadata) {
    return (
      <span className="text-foreground max-w-full font-mono font-semibold break-all">
        {displayVersion}
      </span>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={version}
          className="text-foreground focus-visible:ring-ring/50 cursor-help rounded-sm border-b border-dotted font-mono font-semibold outline-none focus-visible:ring-[3px]"
        >
          {displayVersion}
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-[min(24rem,calc(100vw-2rem))] font-mono break-all">
        {version}
      </TooltipContent>
    </Tooltip>
  )
}
