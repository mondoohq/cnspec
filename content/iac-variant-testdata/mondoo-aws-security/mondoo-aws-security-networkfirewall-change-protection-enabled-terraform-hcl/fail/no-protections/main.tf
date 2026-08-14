# All three protections default to false when omitted.
resource "aws_networkfirewall_firewall" "perimeter" {
  name                = "perimeter"
  firewall_policy_arn = aws_networkfirewall_firewall_policy.perimeter.arn
  vpc_id              = aws_vpc.prod.id

  subnet_mapping {
    subnet_id = aws_subnet.firewall_a.id
  }
}
