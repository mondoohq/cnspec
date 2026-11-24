# Non-compliant: the access string carries the nopass rule.
resource "oci_redis_oci_cache_user" "app" {
  compartment_id = "ocid1.compartment.oc1..example"
  name           = "app"
  description    = "Cache user with no password"
  acl_string     = "on nopass ~* +@all"
  status         = "ON"

  authentication_mode {
    authentication_type = "IAM"
  }
}
