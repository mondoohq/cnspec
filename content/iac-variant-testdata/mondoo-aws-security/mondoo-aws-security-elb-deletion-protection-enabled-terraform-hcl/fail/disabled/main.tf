resource "aws_lb" "fail" {
  name                       = "example-lb"
  enable_deletion_protection = false
}
