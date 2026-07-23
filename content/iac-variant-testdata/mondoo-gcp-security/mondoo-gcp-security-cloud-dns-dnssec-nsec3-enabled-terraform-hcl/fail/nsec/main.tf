resource "google_dns_managed_zone" "fail" {
  name       = "example-zone"
  dns_name   = "example.com."
  visibility = "public"

  dnssec_config {
    state         = "on"
    non_existence = "nsec"
  }
}
