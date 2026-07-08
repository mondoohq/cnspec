resource "google_dns_managed_zone" "pass" {
  name        = "example-zone"
  dns_name    = "example.com."
  description = "public zone with dnssec"
  visibility  = "public"

  dnssec_config {
    state = "on"
  }
}
