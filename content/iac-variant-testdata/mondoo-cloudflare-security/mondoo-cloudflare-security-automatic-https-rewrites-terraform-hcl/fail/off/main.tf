resource "cloudflare_zone_setting" "automatic_https_rewrites" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "automatic_https_rewrites"
  value      = "off"
}
