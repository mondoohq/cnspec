# Fifty attempts before lockout leaves ample room for online password guessing.
resource "alicloud_ram_account_password_policy" "default" {
  minimum_password_length = 14
  max_login_attempts      = 50
}
