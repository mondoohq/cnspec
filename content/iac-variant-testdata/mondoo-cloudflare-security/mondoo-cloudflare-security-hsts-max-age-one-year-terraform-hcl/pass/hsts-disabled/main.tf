resource "cloudflare_zone_setting" "security_header" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = false
      max_age            = 0
      include_subdomains = false
      preload            = false
      nosniff            = true
    }
  }
}
