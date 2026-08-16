resource "aws_lb" "pass" {
  name                       = "example-lb"
  enable_deletion_protection = true
}
