$ErrorActionPreference = 'Stop'; # stop on all errors
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$version  = '7.0.0-alpha1'
$url      = "https://releases.mondoo.com/cnspec/${version}/cnspec_${version}_windows_amd64.zip"
$checksum = '8321ac401d54803cbe0b507c513d7d5ffdcad62477a18ac00739f1c4773d4b92'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url64bit      = $url

  checksum64    = $checksum
  checksumType64= 'sha256' #default is checksumType
}

Install-ChocolateyZipPackage @packageArgs 


