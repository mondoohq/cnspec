# Fourteen characters, and a password may not contain the account name.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  password_policy {
    min_password_length           = 14
    password_not_contain_username = true
  }
}
