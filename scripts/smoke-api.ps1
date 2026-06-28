[CmdletBinding()]
param(
    [string]$BaseUrl = "http://127.0.0.1:34115",
    [int]$TimeoutSec = 10
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0

$DefaultBaseUrl = "http://127.0.0.1:34115"
$BaseUrl = ([string]$BaseUrl).Trim()
if ([string]::IsNullOrWhiteSpace($BaseUrl)) {
    Write-Host "FAIL BaseUrl - value cannot be empty"
    exit 2
}
if ($TimeoutSec -lt 1) {
    Write-Host "FAIL TimeoutSec - value must be at least 1"
    exit 2
}
$BaseUrl = $BaseUrl.TrimEnd("/")

function Get-PropertyValue {
    param(
        [Parameter(Mandatory = $true)][object]$Object,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $property = $Object.PSObject.Properties.Item($Name)
    if ($null -eq $property) {
        return $null
    }
    return $property.Value
}

function Test-ConnectionFailureException {
    param([Parameter(Mandatory = $true)][object]$Exception)

    $response = Get-PropertyValue -Object $Exception -Name "Response"
    if ($null -ne $response) {
        return $false
    }

    $typeName = $Exception.GetType().FullName
    if ($Exception -is [System.Net.WebException]) {
        return $true
    }
    if ($typeName -like "*HttpRequestException*" -or $typeName -like "*TaskCanceledException*") {
        return $true
    }
    if ($Exception.InnerException) {
        return Test-ConnectionFailureException -Exception $Exception.InnerException
    }

    return $false
}

function New-ConnectionFailureException {
    param(
        [Parameter(Mandatory = $true)][string]$Message,
        [Parameter(Mandatory = $true)][object]$InnerException
    )

    $exception = [System.Exception]::new($Message, $InnerException)
    $exception.Data["AgentMeterConnectionFailure"] = $true
    return $exception
}

function Get-HttpStatusCodeFromException {
    param([Parameter(Mandatory = $true)][object]$Exception)

    $response = Get-PropertyValue -Object $Exception -Name "Response"
    if ($null -eq $response) {
        return $null
    }

    $statusCode = Get-PropertyValue -Object $response -Name "StatusCode"
    if ($null -eq $statusCode) {
        return $null
    }
    return [int]$statusCode
}

function Invoke-ApiGet {
    param([Parameter(Mandatory = $true)][string]$Path)

    $requestParams = @{
        Uri = "$BaseUrl$Path"
        Method = "GET"
        Headers = @{ Accept = "application/json" }
        TimeoutSec = $TimeoutSec
    }
    if ($PSVersionTable.PSVersion.Major -lt 6) {
        $requestParams.UseBasicParsing = $true
    }

    try {
        $response = Invoke-WebRequest @requestParams
    } catch {
        if (Test-ConnectionFailureException -Exception $_.Exception) {
            $message = "Cannot reach AgentMeter at $BaseUrl. The backend is expected on 127.0.0.1:34115 ($DefaultBaseUrl); start it or rerun with -BaseUrl."
            throw (New-ConnectionFailureException -Message $message -InnerException $_.Exception)
        }

        $statusCode = Get-HttpStatusCodeFromException -Exception $_.Exception
        if ($null -ne $statusCode) {
            throw "HTTP $statusCode"
        }
        throw $_.Exception.Message
    }

    $status = [int]$response.StatusCode
    if ($status -lt 200 -or $status -ge 300) {
        throw "HTTP $status"
    }

    return [string]$response.Content
}

function Convert-ResponseJson {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Body
    )

    if ([string]::IsNullOrWhiteSpace($Body)) {
        throw "empty response body"
    }

    try {
        return ($Body | ConvertFrom-Json)
    } catch {
        throw "invalid JSON: $($_.Exception.Message)"
    }
}

function Assert-JsonObject {
    param(
        [Parameter(Mandatory = $true)][object]$Value,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if ($null -eq $Value -or $Value -is [System.Array] -or -not ($Value -is [pscustomobject])) {
        throw "expected $Label to be a JSON object"
    }
}

function Get-JsonProperty {
    param(
        [Parameter(Mandatory = $true)][object]$Object,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $property = $Object.PSObject.Properties.Item($Name)
    if ($null -eq $property) {
        throw "missing '$Name'"
    }
    return $property
}

function Test-Number {
    param([object]$Value)

    return (
        $Value -is [byte] -or
        $Value -is [sbyte] -or
        $Value -is [int16] -or
        $Value -is [uint16] -or
        $Value -is [int32] -or
        $Value -is [uint32] -or
        $Value -is [int64] -or
        $Value -is [uint64] -or
        $Value -is [single] -or
        $Value -is [double] -or
        $Value -is [decimal]
    )
}

function Assert-StringProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    if (-not ($value -is [string])) {
        throw "expected '$Name' to be a string"
    }
}

function Assert-NumberProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    if (-not (Test-Number -Value $value)) {
        throw "expected '$Name' to be a number"
    }
}

function Assert-BoolProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    if (-not ($value -is [bool])) {
        throw "expected '$Name' to be a boolean"
    }
}

function Assert-ArrayProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    if (-not ($value -is [System.Array])) {
        throw "expected '$Name' to be an array"
    }
}

function Assert-NullableArrayProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    if ($null -ne $value -and -not ($value -is [System.Array])) {
        throw "expected '$Name' to be an array or null"
    }
}

function Assert-ObjectProperty {
    param([object]$Object, [string]$Name)

    $value = (Get-JsonProperty -Object $Object -Name $Name).Value
    Assert-JsonObject -Value $value -Label "'$Name'"
}

function Assert-TopLevelArray {
    param([Parameter(Mandatory = $true)][string]$Raw)

    if (-not $Raw.TrimStart().StartsWith("[")) {
        throw "expected response to be a JSON array"
    }
}

function Get-FirstArrayItem {
    param(
        [object]$Payload,
        [Parameter(Mandatory = $true)][string]$Raw
    )

    Assert-TopLevelArray -Raw $Raw
    $items = @($Payload)
    if ($items.Count -eq 0) {
        return $null
    }
    return $items[0]
}

function Assert-PrivacySummary {
    param([Parameter(Mandatory = $true)][object]$Summary)

    Assert-JsonObject -Value $Summary -Label "'summary'"
    Assert-NumberProperty -Object $Summary -Name "score"
    Assert-NumberProperty -Object $Summary -Name "total"
    Assert-NumberProperty -Object $Summary -Name "hardened"
    Assert-NumberProperty -Object $Summary -Name "attention"
    Assert-NumberProperty -Object $Summary -Name "implicit"
}

function Assert-PrivacyStatus {
    param(
        [Parameter(Mandatory = $true)][object]$Payload,
        [Parameter(Mandatory = $true)][string]$ExpectedTarget
    )

    Assert-JsonObject -Value $Payload -Label "response"
    Assert-StringProperty -Object $Payload -Name "target"
    Assert-StringProperty -Object $Payload -Name "name"
    Assert-StringProperty -Object $Payload -Name "configPath"
    Assert-BoolProperty -Object $Payload -Name "exists"
    Assert-ArrayProperty -Object $Payload -Name "settings"
    Assert-NullableArrayProperty -Object $Payload -Name "warnings"
    Assert-PrivacySummary -Summary (Get-JsonProperty -Object $Payload -Name "summary").Value

    $target = (Get-JsonProperty -Object $Payload -Name "target").Value
    if ($target -ne $ExpectedTarget) {
        throw "expected 'target' to be '$ExpectedTarget'"
    }
}

function Assert-Settings {
    param([Parameter(Mandatory = $true)][object]$Payload)

    Assert-JsonObject -Value $Payload -Label "response"
    Assert-StringProperty -Object $Payload -Name "sourcePath"
    Assert-ArrayProperty -Object $Payload -Name "sourcePaths"
    Assert-ArrayProperty -Object $Payload -Name "sourceEntries"
    Assert-StringProperty -Object $Payload -Name "defaultSourcePath"
    Assert-ArrayProperty -Object $Payload -Name "defaultSourcePaths"
    Assert-StringProperty -Object $Payload -Name "databasePath"
    Assert-ArrayProperty -Object $Payload -Name "pricingModels"
}

function Assert-Overview {
    param([Parameter(Mandatory = $true)][object]$Payload)

    Assert-JsonObject -Value $Payload -Label "response"
    Assert-NumberProperty -Object $Payload -Name "totalSessions"
    Assert-NumberProperty -Object $Payload -Name "totalTokens"
    Assert-NumberProperty -Object $Payload -Name "totalToolCalls"
    Assert-ArrayProperty -Object $Payload -Name "dailyUsage"
    Assert-ArrayProperty -Object $Payload -Name "modelUsage"
    Assert-ArrayProperty -Object $Payload -Name "agentUsage"
    Assert-ArrayProperty -Object $Payload -Name "recentSessions"
}

function Assert-Sessions {
    param([object]$Payload, [string]$Raw)

    $item = Get-FirstArrayItem -Payload $Payload -Raw $Raw
    if ($null -eq $item) {
        return
    }

    Assert-JsonObject -Value $item -Label "session item"
    Assert-NumberProperty -Object $item -Name "id"
    Assert-StringProperty -Object $item -Name "agentKind"
    Assert-StringProperty -Object $item -Name "sessionKey"
    Assert-StringProperty -Object $item -Name "projectPath"
    Assert-NumberProperty -Object $item -Name "toolCallCount"
    Assert-ObjectProperty -Object $item -Name "tokenUsage"
}

function Assert-Tools {
    param([object]$Payload, [string]$Raw)

    $item = Get-FirstArrayItem -Payload $Payload -Raw $Raw
    if ($null -eq $item) {
        return
    }

    Assert-JsonObject -Value $item -Label "tool item"
    Assert-StringProperty -Object $item -Name "toolName"
    Assert-NumberProperty -Object $item -Name "calls"
    Assert-NumberProperty -Object $item -Name "successCalls"
    Assert-NumberProperty -Object $item -Name "failedCalls"
    Assert-NumberProperty -Object $item -Name "totalDurationMs"
    Assert-NumberProperty -Object $item -Name "avgDurationMs"
}

function Assert-ToolCalls {
    param([object]$Payload, [string]$Raw)

    $item = Get-FirstArrayItem -Payload $Payload -Raw $Raw
    if ($null -eq $item) {
        return
    }

    Assert-JsonObject -Value $item -Label "tool-call item"
    Assert-NumberProperty -Object $item -Name "id"
    Assert-NumberProperty -Object $item -Name "sessionId"
    Assert-StringProperty -Object $item -Name "toolName"
    Assert-StringProperty -Object $item -Name "status"
    Assert-NumberProperty -Object $item -Name "durationMs"
}

function Assert-AuditSummary {
    param([Parameter(Mandatory = $true)][object]$Payload)

    Assert-JsonObject -Value $Payload -Label "response"
    Assert-NumberProperty -Object $Payload -Name "totalFindings"
    Assert-NumberProperty -Object $Payload -Name "criticalFindings"
    Assert-NumberProperty -Object $Payload -Name "highFindings"
    Assert-NumberProperty -Object $Payload -Name "mediumFindings"
    Assert-NumberProperty -Object $Payload -Name "lowFindings"
    Assert-ArrayProperty -Object $Payload -Name "recentFindings"
}

function Assert-AuditFindings {
    param([object]$Payload, [string]$Raw)

    $item = Get-FirstArrayItem -Payload $Payload -Raw $Raw
    if ($null -eq $item) {
        return
    }

    Assert-JsonObject -Value $item -Label "audit finding item"
    Assert-NumberProperty -Object $item -Name "id"
    Assert-StringProperty -Object $item -Name "category"
    Assert-StringProperty -Object $item -Name "severity"
    Assert-StringProperty -Object $item -Name "title"
    Assert-StringProperty -Object $item -Name "command"
}

function Assert-Pricing {
    param([object]$Payload, [string]$Raw)

    $item = Get-FirstArrayItem -Payload $Payload -Raw $Raw
    if ($null -eq $item) {
        return
    }

    Assert-JsonObject -Value $item -Label "pricing item"
    Assert-NumberProperty -Object $item -Name "id"
    Assert-StringProperty -Object $item -Name "model"
    Assert-StringProperty -Object $item -Name "normalizedModel"
    Assert-NumberProperty -Object $item -Name "inputPer1m"
    Assert-NumberProperty -Object $item -Name "cachedInputPer1m"
    Assert-NumberProperty -Object $item -Name "outputPer1m"
    Assert-StringProperty -Object $item -Name "source"
}

$checks = @(
    [pscustomobject]@{ Path = "/api/settings"; Validate = { param($payload, $raw) Assert-Settings -Payload $payload } }
    [pscustomobject]@{ Path = "/api/privacy/codex"; Validate = { param($payload, $raw) Assert-PrivacyStatus -Payload $payload -ExpectedTarget "codex" } }
    [pscustomobject]@{ Path = "/api/privacy/gemini"; Validate = { param($payload, $raw) Assert-PrivacyStatus -Payload $payload -ExpectedTarget "gemini" } }
    [pscustomobject]@{ Path = "/api/privacy/claude"; Validate = { param($payload, $raw) Assert-PrivacyStatus -Payload $payload -ExpectedTarget "claude" } }
    [pscustomobject]@{ Path = "/api/overview"; Validate = { param($payload, $raw) Assert-Overview -Payload $payload } }
    [pscustomobject]@{ Path = "/api/sessions?limit=5"; Validate = { param($payload, $raw) Assert-Sessions -Payload $payload -Raw $raw } }
    [pscustomobject]@{ Path = "/api/tools"; Validate = { param($payload, $raw) Assert-Tools -Payload $payload -Raw $raw } }
    [pscustomobject]@{ Path = "/api/tool-calls?limit=5"; Validate = { param($payload, $raw) Assert-ToolCalls -Payload $payload -Raw $raw } }
    [pscustomobject]@{ Path = "/api/audit/summary"; Validate = { param($payload, $raw) Assert-AuditSummary -Payload $payload } }
    [pscustomobject]@{ Path = "/api/audit/findings?limit=5"; Validate = { param($payload, $raw) Assert-AuditFindings -Payload $payload -Raw $raw } }
    [pscustomobject]@{ Path = "/api/pricing"; Validate = { param($payload, $raw) Assert-Pricing -Payload $payload -Raw $raw } }
)

$passed = 0
$failed = 0
$connectionFailure = $false

foreach ($check in $checks) {
    try {
        $body = Invoke-ApiGet -Path $check.Path
        $payload = Convert-ResponseJson -Path $check.Path -Body $body
        & $check.Validate $payload $body

        Write-Host ("PASS {0}" -f $check.Path)
        $passed++
    } catch {
        $failed++
        $message = $_.Exception.Message
        Write-Host ("FAIL {0} - {1}" -f $check.Path, $message)

        if ($_.Exception.Data.Contains("AgentMeterConnectionFailure") -and $_.Exception.Data["AgentMeterConnectionFailure"]) {
            $connectionFailure = $true
            break
        }
    }
}

if ($failed -gt 0) {
    if ($connectionFailure) {
        Write-Host "API smoke failed: backend connection failed before all endpoints could be checked."
    } else {
        Write-Host ("API smoke failed: {0}/{1} endpoints failed." -f $failed, $checks.Count)
    }
    exit 1
}

Write-Host ("API smoke passed: {0} endpoints checked." -f $passed)
exit 0
