resource "aws_lb" "fail" {
  name                       = "example-lb"
  drop_invalid_header_fields = false
}
