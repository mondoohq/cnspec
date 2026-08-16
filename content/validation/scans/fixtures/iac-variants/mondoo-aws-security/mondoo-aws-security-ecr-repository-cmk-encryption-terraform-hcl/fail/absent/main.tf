# Non-compliant: no encryption_configuration block, so ECR defaults to AES256.
resource "aws_ecr_repository" "fail_example" {
  name = "fail-example"
}
