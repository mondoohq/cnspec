resource "google_container_cluster" "dataplane_v2" {
  name               = "dataplane-v2-cluster"
  location           = "us-central1"
  initial_node_count = 1

  datapath_provider = "ADVANCED_DATAPATH"
}
