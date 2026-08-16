# SSH is accepted only from the corporate egress range.
resource "alicloud_network_acl" "app" {
  vpc_id           = "vpc-example"
  network_acl_name = "app-acl"

  ingress_acl_entries {
    network_acl_entry_name = "allow-ssh-from-office"
    policy                 = "accept"
    protocol               = "tcp"
    port                   = "22/22"
    source_cidr_ip         = "203.0.113.0/24"
    entry_type             = "custom"
  }
}
