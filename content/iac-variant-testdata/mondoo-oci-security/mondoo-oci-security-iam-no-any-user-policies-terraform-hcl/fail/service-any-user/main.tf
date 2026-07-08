# Non-compliant: grants a service to any-user.
resource "oci_identity_policy" "service_any_user" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "service-broad"
  description    = "Broad service access"
  statements = [
    "Allow service any-user to use log-content in tenancy"
  ]
}
