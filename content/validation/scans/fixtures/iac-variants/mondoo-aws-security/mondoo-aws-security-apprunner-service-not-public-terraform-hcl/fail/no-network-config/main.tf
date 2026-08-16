resource "aws_apprunner_service" "default_public" {
  service_name = "default-public"
  source_configuration {
    image_repository {
      image_identifier      = "123456789012.dkr.ecr.us-east-1.amazonaws.com/app:latest"
      image_repository_type = "ECR"
    }
  }
  # No network_configuration block: App Runner defaults is_publicly_accessible = true,
  # so this service is reachable from the public internet and must be flagged.
}
