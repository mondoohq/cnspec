resource "google_dns_managed_zone" "fail" {
  name        = "example-zone"
  dns_name    = "example.com."
  description = "public zone, no dnssec block, default visibility"
}
