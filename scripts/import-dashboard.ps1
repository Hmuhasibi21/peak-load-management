$dsUid = "efin2t6k55urkc"
$json = Get-Content "monitoring\grafana\Peformance Dashboard.json" -Raw -Encoding UTF8
$json = $json -replace [regex]::Escape('${DS_PROMETHEUS}'), $dsUid

$dashboard = $json | ConvertFrom-Json

# Remove export-only fields that cause import issues
if ($dashboard.PSObject.Properties["__inputs"]) { $dashboard.PSObject.Properties.Remove("__inputs") }
if ($dashboard.PSObject.Properties["__elements"]) { $dashboard.PSObject.Properties.Remove("__elements") }
if ($dashboard.PSObject.Properties["__requires"]) { $dashboard.PSObject.Properties.Remove("__requires") }

# Set id to null for new import
$dashboard.id = $null

$body = [ordered]@{
    dashboard = $dashboard
    overwrite = $true
} | ConvertTo-Json -Depth 100

# Write to file with UTF8 no BOM
[System.IO.File]::WriteAllText("$PWD\import-payload.json", $body, [System.Text.UTF8Encoding]::new($false))

$result = curl.exe -s -X POST "http://admin:admin123@localhost:3000/api/dashboards/db" -H "Content-Type: application/json" -d "@import-payload.json"
Write-Output $result
