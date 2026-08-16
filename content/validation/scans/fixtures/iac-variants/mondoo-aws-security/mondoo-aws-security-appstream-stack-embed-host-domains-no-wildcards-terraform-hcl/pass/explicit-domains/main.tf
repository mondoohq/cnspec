# Compliant: embed host domains are explicit with no wildcards.
resource "aws_appstream_stack" "pass_example" {
  name = "example-stack"

  embed_host_domains = ["example.com", "app.example.com"]
}
