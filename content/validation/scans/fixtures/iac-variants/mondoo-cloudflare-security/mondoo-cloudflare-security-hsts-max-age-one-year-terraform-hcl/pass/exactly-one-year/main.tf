resource "cloudflare_zone_setting" "security_header" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      max_age            = 31536000
      include_subdomains = true
      preload            = false
      nosniff            = true
    }
  }
}
