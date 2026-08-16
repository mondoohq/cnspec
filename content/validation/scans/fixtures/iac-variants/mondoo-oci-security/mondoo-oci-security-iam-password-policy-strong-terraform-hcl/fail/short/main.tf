# Non-compliant: minimum length below 14.
resource "oci_identity_authentication_policy" "weak_len" {
  compartment_id = "ocid1.tenancy.oc1..aaaaaaaaexampletenancy"

  password_policy {
    minimum_password_length          = 8
    is_uppercase_characters_required = true
    is_lowercase_characters_required = true
    is_numeric_characters_required   = true
    is_special_characters_required   = true
    is_username_containment_allowed  = false
  }
}
