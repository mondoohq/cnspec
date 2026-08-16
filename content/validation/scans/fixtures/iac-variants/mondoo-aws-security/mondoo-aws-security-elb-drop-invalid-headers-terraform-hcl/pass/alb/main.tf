resource "aws_alb" "pass" {
  name                       = "example-alb"
  drop_invalid_header_fields = true
}
