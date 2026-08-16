# The entry never names port 22, but 1/1024 spans it, so SSH is accepted from the
# whole internet at the subnet boundary in front of every security group.
resource "alicloud_network_acl" "app" {
  vpc_id           = "vpc-example"
  network_acl_name = "app-acl"

  ingress_acl_entries {
    network_acl_entry_name = "allow-privileged-ports"
    policy                 = "accept"
    protocol               = "tcp"
    port                   = "1/1024"
    source_cidr_ip         = "0.0.0.0/0"
    entry_type             = "custom"
  }
}
