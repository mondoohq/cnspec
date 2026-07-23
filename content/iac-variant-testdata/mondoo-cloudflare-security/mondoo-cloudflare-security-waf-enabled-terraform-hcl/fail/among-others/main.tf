resource "cloudflare_zone_setting" "always_use_https" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "always_use_https"
  value      = "on"
}

resource "cloudflare_zone_setting" "waf" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "waf"
  value      = "off"
}
