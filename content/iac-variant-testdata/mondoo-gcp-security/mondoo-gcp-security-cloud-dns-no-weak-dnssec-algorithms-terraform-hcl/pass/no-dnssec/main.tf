resource "google_dns_managed_zone" "pass" {
  name       = "example-zone"
  dns_name   = "example.com."
  visibility = "public"
}
