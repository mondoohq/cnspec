resource "portainer_environment" "prod_agent" {
  name        = "prod-agent"
  type        = 2
  address     = "tcp://agent.prod.example.com:9001"
  tls_enabled = true
}
