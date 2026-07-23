resource "openstack_fw_policy_v2" "default" {
  name    = "default-policy"
  audited = true

  rules = [
    openstack_fw_rule_v2.allow_https.id,
  ]
}

resource "openstack_fw_rule_v2" "allow_https" {
  name             = "allow-https"
  protocol         = "tcp"
  action           = "allow"
  destination_port = "443"
  enabled          = true
}
