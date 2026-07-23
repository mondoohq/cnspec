# Non-compliant: endorse statement grants cross-tenancy access.
resource "oci_identity_policy" "cross_tenant" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "cross-tenancy"
  description    = "Cross-tenant sharing"
  statements = [
    "Endorse group Auditors to read buckets in any-tenancy"
  ]
}
