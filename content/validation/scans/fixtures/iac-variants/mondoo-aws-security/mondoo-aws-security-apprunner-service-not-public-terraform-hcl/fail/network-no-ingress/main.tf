# Non-compliant: a network_configuration block exists but omits ingress_configuration,
# so is_publicly_accessible defaults to true. The check must require the ingress block to
# exist and explicitly set false, not pass vacuously.
resource "aws_apprunner_service" "partial" {
  service_name = "partial"
  source_configuration {
    image_repository {
      image_identifier      = "123456789012.dkr.ecr.us-east-1.amazonaws.com/app:latest"
      image_repository_type = "ECR"
    }
  }
  network_configuration {
    egress_configuration {
      egress_type = "DEFAULT"
    }
  }
}
