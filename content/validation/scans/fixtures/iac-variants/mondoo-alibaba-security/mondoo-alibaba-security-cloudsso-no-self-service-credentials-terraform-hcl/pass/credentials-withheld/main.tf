# Users cannot mint credentials that outlive their session.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  login_preference {
    allow_user_to_get_credentials = false
  }
}
