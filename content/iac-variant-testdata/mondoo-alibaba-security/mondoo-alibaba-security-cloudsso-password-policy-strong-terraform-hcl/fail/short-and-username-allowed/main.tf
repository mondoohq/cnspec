# Eight characters, and a password may contain the account name. Both are settable
# through SetPasswordPolicy, unlike the character-class flags the directory reports.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  password_policy {
    min_password_length           = 8
    password_not_contain_username = false
  }
}
