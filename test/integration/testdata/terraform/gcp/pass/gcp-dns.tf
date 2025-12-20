resource "google_dns_managed_zone" "foo" {
	name     = "foobar"
	dns_name = "foo.bar."
	
	dnssec_config {
		state         = "on"
		non_existence = "nsec3"
	}
}
	
data "google_dns_keys" "foo_dns_keys" {
	managed_zone = google_dns_managed_zone.foo.id
	zone_signing_keys {
		algorithm = "rsasha512"
	}
}
	
output "foo_dns_ds_record" {
	description = "DS record of the foo subdomain."
	value       = data.google_dns_keys.foo_dns_keys.key_signing_keys[0].ds_record
}

resource "google_dns_managed_zone" "good_example" {
  name        = "example-zone"
  dns_name    = "example-${random_id.rnd.hex}.com."
  description = "Example DNS zone"
  labels = {
    foo = "bar"
  }
  dnssec_config {
    state = "on"
  }
}

resource "random_id" "rnd" {
  byte_length = 4
}