# Compliant: direct internet access is disabled.
resource "aws_sagemaker_notebook_instance" "pass_example" {
  name                   = "example-notebook"
  instance_type          = "ml.t3.medium"
  role_arn               = "arn:aws:iam::123456789012:role/example"
  direct_internet_access = "Disabled"
}
