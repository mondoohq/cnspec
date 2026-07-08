# Non-compliant: kms_key_id resolves to an empty string.
resource "aws_sagemaker_notebook_instance" "fail_example" {
  name          = "example-notebook"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
  kms_key_id    = ""
}
