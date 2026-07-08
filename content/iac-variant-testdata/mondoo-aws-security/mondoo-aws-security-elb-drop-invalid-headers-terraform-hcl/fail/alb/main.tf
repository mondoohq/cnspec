resource "aws_alb" "fail" {
  name                       = "example-alb"
  drop_invalid_header_fields = false
}
