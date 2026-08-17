resource "stackit_application_load_balancer" "example" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  region     = "eu01"
  name       = "example-alb"
  plan_id    = "p10"

  listeners = [{
    name     = "https"
    port     = 443
    protocol = "PROTOCOL_HTTPS"
    http = {
      hosts = [{
        host = "*"
        rules = [{
          target_pool = "example-target-pool"
        }]
      }]
    }
    https = {
      certificate_config = {
        certificate_ids = ["9f8e7d6c-5b4a-4392-8172-6a5b4c3d2e1f"]
      }
    }
  }]

  networks = [{
    network_id = "5f6a1c22-7d3b-4f18-9a2c-0b7e8d9f1a2b"
    role       = "ROLE_LISTENERS_AND_TARGETS"
  }]

  target_pools = [{
    name        = "example-target-pool"
    target_port = 8443
    targets = [{
      ip = "10.0.1.20"
    }]
  }]

  options = {
    ephemeral_address = true
  }

  disable_target_security_group_assignment = true
}
