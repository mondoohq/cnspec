# Non-compliant: grants access to any-user (every authenticated principal).
resource "oci_identity_policy" "any_user" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "public-read"
  description    = "Broad access"
  statements = [
    "Allow any-user to manage objects in compartment Data"
  ]
}
