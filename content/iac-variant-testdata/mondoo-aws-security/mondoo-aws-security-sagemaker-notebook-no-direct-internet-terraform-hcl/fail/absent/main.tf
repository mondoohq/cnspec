# Non-compliant: direct_internet_access is omitted, defaulting to Enabled.
resource "aws_sagemaker_notebook_instance" "fail_example" {
  name          = "example-notebook"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
}
