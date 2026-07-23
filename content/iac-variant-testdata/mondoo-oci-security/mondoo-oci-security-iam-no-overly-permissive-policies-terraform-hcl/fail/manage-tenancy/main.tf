# Non-compliant: grants manage all-resources across the entire tenancy.
resource "oci_identity_policy" "tenant_admin" {
  compartment_id = "ocid1.tenancy.oc1..aaaaaaaaexampletenancy"
  name           = "tenancy-admins"
  description    = "Tenancy-wide administration"
  statements = [
    "Allow group Admins to manage all-resources in tenancy"
  ]
}
