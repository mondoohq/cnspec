# Compliant: IP access is restricted to corporate CIDR ranges.
resource "aws_workspacesweb_ip_access_settings" "pass_example" {
  display_name = "corporate-only"

  ip_rule {
    ip_range    = "203.0.113.0/24"
    description = "Corporate egress"
  }

  ip_rule {
    ip_range    = "198.51.100.0/24"
    description = "VPN egress"
  }
}
