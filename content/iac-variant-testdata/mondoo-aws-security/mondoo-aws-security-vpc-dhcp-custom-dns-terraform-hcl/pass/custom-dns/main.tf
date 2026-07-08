resource "aws_vpc_dhcp_options" "example" {
  domain_name         = "internal.example.com"
  domain_name_servers = ["10.0.0.2", "10.0.0.3"]
  ntp_servers         = ["10.0.0.4"]
}
