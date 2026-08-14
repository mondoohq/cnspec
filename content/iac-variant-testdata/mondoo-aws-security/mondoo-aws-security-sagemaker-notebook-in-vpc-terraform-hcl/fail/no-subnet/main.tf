# With no subnet the notebook runs on the SageMaker-managed network, outside
# the VPC's segmentation, flow logs, and egress filtering.
resource "aws_sagemaker_notebook_instance" "research" {
  name          = "research-notebook"
  role_arn      = aws_iam_role.sagemaker.arn
  instance_type = "ml.t3.medium"
}
