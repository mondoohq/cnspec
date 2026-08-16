# Non-compliant: embed host domains include a wildcard entry.
resource "aws_appstream_stack" "fail_example" {
  name = "example-stack"

  embed_host_domains = ["*.example.com", "app.example.com"]
}
