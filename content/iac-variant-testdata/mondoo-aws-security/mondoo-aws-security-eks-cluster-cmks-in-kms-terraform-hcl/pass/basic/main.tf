# Compliant: EKS cluster encrypts secrets with a customer-managed KMS key.
resource "aws_eks_cluster" "pass_example" {
  name     = "pass_example_cluster"
  role_arn = var.cluster_arn

  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = var.kms_arn
    }
  }

  vpc_config {
    endpoint_public_access  = false
    endpoint_private_access = true
  }
}
