resource "google_dns_managed_zone" "pass" {
  name       = "example-zone"
  dns_name   = "example.com."
  visibility = "public"

  dnssec_config {
    state         = "on"
    non_existence = "nsec3"
  }
}
