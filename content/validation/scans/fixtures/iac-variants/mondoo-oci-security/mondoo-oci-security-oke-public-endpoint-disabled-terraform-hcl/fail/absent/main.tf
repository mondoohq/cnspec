# Non-compliant: cluster omits endpoint_config; Oracle defaults the API endpoint
# to a public IP.
resource "oci_containerengine_cluster" "prod" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "prod-oke"
  vcn_id             = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"
}
