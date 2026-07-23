# Non-compliant: a counted workgroup with no configuration block at all.
resource "aws_athena_workgroup" "fleet" {
  count = 2
  name  = "fleet-${count.index}"
}
