# Compliant: supported Valkey 8.1 engine version.
resource "oci_redis_redis_cluster" "cache" {
  compartment_id     = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name       = "app-cache"
  node_count         = 3
  node_memory_in_gbs = 2
  software_version   = "VALKEY_8_1"
  subnet_id          = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
}
