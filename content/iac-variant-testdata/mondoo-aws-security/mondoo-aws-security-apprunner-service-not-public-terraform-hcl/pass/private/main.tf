resource "aws_apprunner_service" "internal" {
  service_name = "internal"
  source_configuration {
    image_repository {
      image_identifier      = "123456789012.dkr.ecr.us-east-1.amazonaws.com/app:latest"
      image_repository_type = "ECR"
    }
  }
  network_configuration {
    ingress_configuration {
      is_publicly_accessible = false
    }
  }
}
