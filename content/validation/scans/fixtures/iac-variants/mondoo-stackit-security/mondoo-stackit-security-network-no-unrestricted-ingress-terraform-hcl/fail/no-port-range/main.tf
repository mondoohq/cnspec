resource "stackit_security_group_rule" "any" {
  project_id        = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  security_group_id = "b2c3d4e5-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
  direction         = "ingress"
  ether_type        = "IPv4"
  ip_range          = "0.0.0.0/0"
  protocol = {
    name = "tcp"
  }
}
