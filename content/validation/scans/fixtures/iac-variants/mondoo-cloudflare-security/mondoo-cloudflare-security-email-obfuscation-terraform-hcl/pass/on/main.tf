resource "cloudflare_zone_setting" "email_obfuscation" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "email_obfuscation"
  value      = "on"
}
