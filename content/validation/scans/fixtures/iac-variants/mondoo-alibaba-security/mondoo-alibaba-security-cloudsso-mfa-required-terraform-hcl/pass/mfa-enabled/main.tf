# Every sign-in to this directory needs a second factor.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name            = "workforce"
  mfa_authentication_status = "Enabled"
}
