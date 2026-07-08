resource "cloudflare_zone_setting" "opportunistic_encryption" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "opportunistic_encryption"
  value      = "off"
}
