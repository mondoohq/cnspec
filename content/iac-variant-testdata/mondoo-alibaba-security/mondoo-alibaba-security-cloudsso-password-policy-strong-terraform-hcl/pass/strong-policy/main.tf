# 14 characters drawn from all four character classes.
resource "alicloud_cloud_sso_directory" "workforce" {
  directory_name = "workforce"

  password_policy {
    min_password_length      = 14
    require_upper_case_chars = true
    require_lower_case_chars = true
    require_numbers          = true
    require_symbols          = true
  }
}
