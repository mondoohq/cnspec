$ErrorActionPreference = 'Stop'; # stop on all errors
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$version  = '7.0.3'
$url      = "https://releases.mondoo.com/cnspec/${version}/cnspec_${version}_windows_amd64.zip"
$checksum = 'ce0e976e0f073d7155e3f0d355816658f8691bba81ec8614e480cc05a96e6a20'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url64bit      = $url

  checksum64    = $checksum
  checksumType64= 'sha256' #default is checksumType
}

Install-ChocolateyZipPackage @packageArgs 


