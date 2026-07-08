resource "aws_ecs_task_definition" "pass" {
  family                = "service"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  volume {
    name = "efs-vol"

    efs_volume_configuration {
      file_system_id     = "fs-12345678"
      transit_encryption = "ENABLED"
    }
  }
}
