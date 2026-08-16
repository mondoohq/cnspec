resource "cloudflare_zone_dnssec" "this" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  status  = "active"
}
