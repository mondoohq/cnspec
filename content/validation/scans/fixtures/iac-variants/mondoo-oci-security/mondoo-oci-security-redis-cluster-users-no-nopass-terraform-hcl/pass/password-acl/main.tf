# Compliant: the access string scopes keys and commands and carries no nopass rule.
resource "oci_redis_oci_cache_user" "app" {
  compartment_id = "ocid1.compartment.oc1..example"
  name           = "app"
  description    = "Application cache user"
  acl_string     = "~app:* +@read +@write"
  status         = "ON"

  authentication_mode {
    authentication_type = "PASSWORD"
    hashed_passwords    = ["a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"]
  }
}
