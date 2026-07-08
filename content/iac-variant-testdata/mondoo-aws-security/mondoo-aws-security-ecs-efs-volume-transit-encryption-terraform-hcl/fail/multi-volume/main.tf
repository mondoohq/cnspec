# Non-compliant: task definition with two volumes; one has transit encryption
# DISABLED. .all() over volume blocks must still fail.
resource "aws_ecs_task_definition" "fail" {
  family                = "service"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  volume {
    name = "efs-vol-a"

    efs_volume_configuration {
      file_system_id     = "fs-11111111"
      transit_encryption = "ENABLED"
    }
  }

  volume {
    name = "efs-vol-b"

    efs_volume_configuration {
      file_system_id     = "fs-22222222"
      transit_encryption = "DISABLED"
    }
  }
}
