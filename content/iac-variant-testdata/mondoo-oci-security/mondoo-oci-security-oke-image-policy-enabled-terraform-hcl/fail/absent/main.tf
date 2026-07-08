# Non-compliant: cluster omits image_policy_config, so image signature
# verification is not enforced.
resource "oci_containerengine_cluster" "prod" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "prod-oke"
  vcn_id             = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
}
