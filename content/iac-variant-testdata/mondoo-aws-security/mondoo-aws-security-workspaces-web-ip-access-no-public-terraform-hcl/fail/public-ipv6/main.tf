# Non-compliant: IP access allows the entire IPv6 internet.
resource "aws_workspacesweb_ip_access_settings" "fail_example" {
  display_name = "wide-open-v6"

  ip_rule {
    ip_range    = "203.0.113.0/24"
    description = "Corporate egress"
  }

  ip_rule {
    ip_range    = "::/0"
    description = "Allow all IPv6"
  }
}
