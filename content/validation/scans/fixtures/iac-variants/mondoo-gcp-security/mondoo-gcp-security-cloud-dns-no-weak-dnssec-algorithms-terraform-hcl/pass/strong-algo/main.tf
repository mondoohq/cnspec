resource "google_dns_managed_zone" "pass" {
  name       = "example-zone"
  dns_name   = "example.com."
  visibility = "public"

  dnssec_config {
    state = "on"

    default_key_specs {
      algorithm  = "ecdsap256sha256"
      key_type   = "keySigning"
      key_length = 256
    }
    default_key_specs {
      algorithm  = "ecdsap256sha256"
      key_type   = "zoneSigning"
      key_length = 256
    }
  }
}
