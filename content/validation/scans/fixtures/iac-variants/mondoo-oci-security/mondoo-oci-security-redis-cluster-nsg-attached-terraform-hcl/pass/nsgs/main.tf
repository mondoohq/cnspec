# Compliant: Redis cluster attaches a network security group.
resource "oci_redis_redis_cluster" "cache" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name       = "app-cache"
  node_count         = 3
  node_memory_in_gbs = 2
  software_version   = "VALKEY_7_2"
  subnet_id          = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  nsg_ids            = ["ocid1.networksecuritygroup.oc1.iad.aaaaaaaaexamplensg"]
}
