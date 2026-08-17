resource "stackit_ske_cluster" "prod" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name       = "prod"

  node_pools = [{
    name               = "default"
    machine_type       = "c1.2"
    minimum            = 2
    maximum            = 3
    availability_zones = ["eu01-1"]
  }]

  maintenance = {
    enable_kubernetes_version_updates = false
    start                             = "01:00:00Z"
    end                               = "03:00:00Z"
  }
}
