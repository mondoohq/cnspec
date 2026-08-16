# A logging configuration with no destinations records nothing, so alerts and
# flow records from the firewall are lost.
resource "aws_networkfirewall_logging_configuration" "perimeter" {
  firewall_arn = aws_networkfirewall_firewall.perimeter.arn

  logging_configuration {
  }
}
