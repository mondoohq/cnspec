resource "cloudflare_zone_setting" "security_level" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "security_level"
  value      = "essentially_off"
}
