# Non-compliant: one of two users carries NOPASS, in upper case and mid-string.
resource "oci_redis_oci_cache_user" "good" {
  compartment_id = "ocid1.compartment.oc1..example"
  name           = "good"
  description    = "Cache user with a scoped access string"
  acl_string     = "~app:* +@read"
  status         = "ON"

  authentication_mode {
    authentication_type = "IAM"
  }
}

resource "oci_redis_oci_cache_user" "bad" {
  compartment_id = "ocid1.compartment.oc1..example"
  name           = "bad"
  description    = "Cache user with no password, written in upper case"
  acl_string     = "~* NOPASS +@read"
  status         = "ON"

  authentication_mode {
    authentication_type = "IAM"
  }
}
