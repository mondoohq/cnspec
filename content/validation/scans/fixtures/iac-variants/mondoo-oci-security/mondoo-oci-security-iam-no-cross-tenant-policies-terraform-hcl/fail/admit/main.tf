# Non-compliant: admit statement accepts cross-tenancy access.
resource "oci_identity_policy" "cross_tenant" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "cross-tenancy-admit"
  description    = "Cross-tenant sharing"
  statements = [
    "Admit group Auditors of tenancy AuditTenant to read buckets in compartment Data"
  ]
}
