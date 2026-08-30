resource "portainer_environment" "edge_agent" {
  name                = "branch-edge"
  type                = 6
  environment_address = "tcp://edge.branch.example.com:9001"
  tls_enabled         = false
}
