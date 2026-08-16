resource "alicloud_ram_account_password_policy" "default" {
  minimum_password_length      = 14
  require_lowercase_characters = true
  require_uppercase_characters = true
  require_numbers              = true
  require_symbols              = true
  max_login_attempts           = 5
  password_reuse_prevention    = 5
  max_password_age             = 90
}
