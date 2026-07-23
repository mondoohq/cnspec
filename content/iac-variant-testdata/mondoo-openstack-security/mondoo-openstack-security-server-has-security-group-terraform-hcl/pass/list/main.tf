# Compliant: the instance is attached to a security group.
resource "openstack_compute_instance_v2" "web" {
  name            = "web-01"
  image_id        = "a1b2c3d4-1234-5678-90ab-cdef01234567"
  flavor_id       = "3"
  key_pair        = "ops-key"
  security_groups = ["web-sg"]

  network {
    name = "internal"
  }
}
