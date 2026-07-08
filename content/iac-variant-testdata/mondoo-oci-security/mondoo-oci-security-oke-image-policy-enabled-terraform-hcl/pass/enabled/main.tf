# Compliant: image verification policy is enabled on the cluster.
resource "oci_containerengine_cluster" "prod" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "prod-oke"
  vcn_id             = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"

  image_policy_config {
    is_policy_enabled = true
    key_details {
      kms_key_id = "ocid1.key.oc1.iad.aaaaaaaaexamplekey"
    }
  }
}
