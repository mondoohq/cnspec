# A workgroup with no configuration block enforces nothing at all.
resource "aws_athena_workgroup" "analytics" {
  name = "analytics"
}
