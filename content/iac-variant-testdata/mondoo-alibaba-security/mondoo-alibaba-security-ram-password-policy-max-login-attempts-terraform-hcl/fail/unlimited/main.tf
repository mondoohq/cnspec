# 0 means no lockout at all, so guessing is unbounded.
resource "alicloud_ram_account_password_policy" "default" {
  minimum_password_length = 14
  max_login_attempts      = 0
}
