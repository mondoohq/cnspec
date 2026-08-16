# "Byp" leaves the service to decide which sign-ins look risky, so a sign-in from a
# familiar network or device never gets a second-factor prompt.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name            = "workforce"
  mfa_authentication_status = "Byp"
}
