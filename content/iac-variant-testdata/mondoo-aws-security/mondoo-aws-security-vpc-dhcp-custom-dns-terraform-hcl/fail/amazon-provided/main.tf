resource "aws_vpc_dhcp_options" "example" {
  domain_name         = "ec2.internal"
  domain_name_servers = ["AmazonProvidedDNS"]
}
