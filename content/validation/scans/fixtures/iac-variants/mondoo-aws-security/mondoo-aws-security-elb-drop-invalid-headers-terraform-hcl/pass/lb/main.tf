resource "aws_lb" "pass" {
  name                       = "example-lb"
  drop_invalid_header_fields = true
}
