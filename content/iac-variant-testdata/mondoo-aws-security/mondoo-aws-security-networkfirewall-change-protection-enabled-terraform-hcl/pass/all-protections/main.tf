resource "aws_networkfirewall_firewall" "perimeter" {
  name                              = "perimeter"
  firewall_policy_arn               = aws_networkfirewall_firewall_policy.perimeter.arn
  vpc_id                            = aws_vpc.prod.id
  delete_protection                 = true
  firewall_policy_change_protection = true
  subnet_change_protection          = true

  subnet_mapping {
    subnet_id = aws_subnet.firewall_a.id
  }
}
