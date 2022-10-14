function global:au_GetLatest {
  $response = Invoke-WebRequest -Uri https://api.github.com/repos/mondoohq/cnspec/releases/latest
  $release = ConvertFrom-Json $response.Content
  $regex   = '_windows_amd64.zip$'
  $version = $release.tag_name -replace '^v',''
  $downloaduUrl = $release.assets | ? name -match $regex | select -First 1 -expand browser_download_url
  $checksumsUrl  = $release.assets | ? name -match '_SHA256SUMS$' | select -First 1 -expand browser_download_url
  return @{ 
    Version = $version
    GithubDownloadUrl = $downloadUrl 
    ChecksumsUrl = $checksumsUrl
  }
}

function global:au_BeforeUpdate {
  $response = Invoke-WebRequest -Uri $Latest.ChecksumsUrl
  $rawCsv = ([System.Text.Encoding]::ASCII.GetString($response.Content)) -replace '  ',','
  $checksum = ConvertFrom-Csv -Header checksum,name $rawCsv | ? name -match '_windows_amd64.zip' | Select -First 1 -expand checksum

  $Latest.Checksum64 = $checksum
}

function global:au_SearchReplace {
  @{
    "tools\chocolateyInstall.ps1" = @{
      "(^[$]version\s*=\s*)('.*')" = "`$1'$($Latest.Version)'"
      "(^[$]checksum\s*=\s*)('.*')" = "`$1'$($Latest.Checksum64)'"
    }
  }
}

update -ChecksumFor none
