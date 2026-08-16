# Non-compliant: node pool omits node_config_details, so no NSGs are attached.
resource "oci_containerengine_node_pool" "workers" {
  cluster_id         = "ocid1.cluster.oc1.iad.aaaaaaaaexamplecluster"
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  kubernetes_version = "v1.29.1"
  name               = "workers"
  node_shape         = "VM.Standard.E4.Flex"
}
