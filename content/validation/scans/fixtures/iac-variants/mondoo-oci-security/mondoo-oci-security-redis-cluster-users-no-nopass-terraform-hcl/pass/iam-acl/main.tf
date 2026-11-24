# Compliant: nopass appears only inside a key pattern, not as a rule.
resource "oci_redis_oci_cache_user" "reader" {
  compartment_id = "ocid1.compartment.oc1..example"
  name           = "reader"
  description    = "Read-only cache user whose key prefix contains the word nopass"
  acl_string     = "on ~nopass-cache:* +@read"
  status         = "ON"

  authentication_mode {
    authentication_type = "IAM"
  }
}
