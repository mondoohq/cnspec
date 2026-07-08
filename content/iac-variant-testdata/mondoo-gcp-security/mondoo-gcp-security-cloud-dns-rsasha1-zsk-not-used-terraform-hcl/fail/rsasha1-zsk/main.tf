resource "google_dns_managed_zone" "fail" {
  name       = "example-zone"
  dns_name   = "example.com."
  visibility = "public"

  dnssec_config {
    state = "on"

    default_key_specs {
      algorithm  = "rsasha1"
      key_type   = "zoneSigning"
      key_length = 1024
    }
  }
}
