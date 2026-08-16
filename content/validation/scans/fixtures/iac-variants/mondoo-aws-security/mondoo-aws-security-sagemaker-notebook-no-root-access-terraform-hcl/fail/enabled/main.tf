# Non-compliant: root access enabled on the notebook instance.
resource "aws_sagemaker_notebook_instance" "fail_example" {
  name          = "example"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
  root_access   = "Enabled"
}
