# Non-compliant: repository uses AES256 rather than KMS.
resource "aws_ecr_repository" "fail_example" {
  name = "fail-example"

  encryption_configuration {
    encryption_type = "AES256"
  }
}
