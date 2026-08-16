# Non-compliant: domain has no customer managed encryption key.
resource "aws_codeartifact_domain" "fail_example" {
  domain = "example-domain"
}
