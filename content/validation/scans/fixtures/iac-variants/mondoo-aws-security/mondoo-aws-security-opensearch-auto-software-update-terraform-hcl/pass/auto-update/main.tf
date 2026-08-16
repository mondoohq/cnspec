resource "aws_opensearch_domain" "example" {
  domain_name = "example"

  software_update_options {
    auto_software_update_enabled = true
  }
}
