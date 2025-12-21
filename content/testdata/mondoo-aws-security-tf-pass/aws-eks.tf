resource "aws_eks_cluster" "good_example" {
  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = var.kms_arn
    }
  }

  name     = "good_example_cluster"
  role_arn = var.cluster_arn
  vpc_config {
    endpoint_public_access = false
    endpoint_private_access = true
  }
}

// VOC with restricted public access CIDRs
# resource "aws_eks_cluster" "good_example" {
#   // other config
#   name = "good_example_cluster"
#   role_arn = var.cluster_arn
#   encryption_config {
#     resources = ["secrets"]
#     provider {
#       key_arn = var.kms_arn
#     }
#   }
#   vpc_config {
#     endpoint_private_access = true
#     endpoint_public_access = true
#     public_access_cidrs = ["10.2.0.1/8"]
#   }
# }