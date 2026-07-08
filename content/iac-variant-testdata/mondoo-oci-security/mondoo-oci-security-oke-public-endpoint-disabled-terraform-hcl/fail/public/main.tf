# Non-compliant: cluster API endpoint is explicitly given a public IP.
resource "oci_containerengine_cluster" "prod" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "prod-oke"
  vcn_id             = "ocid1.vcn.oc1.iad.aaaaaaaaexamplevcn"

  endpoint_config {
    subnet_id            = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
    is_public_ip_enabled = true
  }
}
