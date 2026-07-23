# Non-compliant: an aws_kms_external_key is present (external key material).
# The aws_kms_key satisfies the variant filter so the check runs.
resource "aws_kms_key" "pass_example" {
  description = "standard key"
}

resource "aws_kms_external_key" "fail_example" {
  description = "external key material"
}
