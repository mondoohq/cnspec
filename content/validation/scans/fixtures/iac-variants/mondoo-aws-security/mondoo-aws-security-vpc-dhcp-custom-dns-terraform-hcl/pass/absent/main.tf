resource "aws_vpc_dhcp_options" "example" {
  domain_name = "internal.example.com"
  ntp_servers = ["10.0.0.4"]
}
