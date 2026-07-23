resource "cloudflare_zone_setting" "tls_1_3" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "tls_1_3"
  value      = "off"
}
