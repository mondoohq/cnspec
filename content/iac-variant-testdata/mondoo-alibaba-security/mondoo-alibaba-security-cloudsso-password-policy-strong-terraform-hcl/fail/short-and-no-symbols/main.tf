# Eight characters and no symbol requirement leaves the keyspace small enough for
# offline cracking to be practical.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  password_policy {
    min_password_length      = 8
    require_upper_case_chars = true
    require_lower_case_chars = true
    require_numbers          = true
    require_symbols          = false
  }
}
