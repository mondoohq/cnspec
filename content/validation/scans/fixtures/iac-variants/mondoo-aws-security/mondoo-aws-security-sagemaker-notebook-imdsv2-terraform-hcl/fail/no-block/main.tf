# Non-compliant: no instance_metadata_service_configuration block, so IMDSv1 is allowed.
resource "aws_sagemaker_notebook_instance" "fail_example" {
  name          = "example-notebook"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
}
