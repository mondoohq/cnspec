# Compliant: strong password policy (>=14 chars, all character classes, no username containment).
resource "oci_identity_authentication_policy" "compliant" {
  compartment_id = "ocid1.tenancy.oc1..aaaaaaaaexampletenancy"

  password_policy {
    minimum_password_length          = 14
    is_uppercase_characters_required = true
    is_lowercase_characters_required = true
    is_numeric_characters_required   = true
    is_special_characters_required   = true
    is_username_containment_allowed  = false
  }
}
