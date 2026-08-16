# A signed-in user can mint long-lived credentials, converting a bounded session
# into standing programmatic access.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  login_preference {
    allow_user_to_get_credentials = true
  }
}
