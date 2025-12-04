resource "google_dns_managed_zone" "example-zone" {
  name        = "example-zone"
  dns_name    = "example-${random_id.rnd.hex}.com."
  description = "Example DNS zone"
  labels = {
    foo = "bar"
  }

  dnssec_config {
    state = "on"

    default_key_specs {
      algorithm  = "rsasha256"
      key_type   = "keySigning"
      key_length = 2048
    }

    default_key_specs {
      algorithm  = "rsasha256"
      key_type   = "zoneSigning"
      key_length = 1024
    }
  }
}

resource "random_id" "rnd" {
  byte_length = 4
}