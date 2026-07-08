# Non-compliant: IP access allows the entire IPv4 internet.
resource "aws_workspacesweb_ip_access_settings" "fail_example" {
  display_name = "wide-open"

  ip_rule {
    ip_range    = "0.0.0.0/0"
    description = "Allow all"
  }
}
