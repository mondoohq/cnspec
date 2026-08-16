resource "cloudflare_zone_setting" "min_tls_version" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "min_tls_version"
  value      = "1.2"
}
