# Non-compliant: node_config_details is present but nsg_ids is an empty list.
resource "oci_containerengine_node_pool" "workers" {
  cluster_id         = "ocid1.cluster.oc1.iad.aaaaaaaaexamplecluster"
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "workers"
  node_shape         = "VM.Standard.E4.Flex"

  node_config_details {
    size    = 3
    nsg_ids = []
    placement_configs {
      availability_domain = "Uocm:PHX-AD-1"
      subnet_id           = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
    }
  }
}
