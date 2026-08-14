# Dropping one character class shrinks the keyspace an attacker has to search.
resource "alicloud_ram_account_password_policy" "default" {
  minimum_password_length      = 14
  require_lowercase_characters = true
  require_uppercase_characters = true
  require_numbers              = true
  require_symbols              = false
  max_login_attempts           = 5
}
