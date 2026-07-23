# Compliant: EFS volume with transit encryption ENABLED, generated via a
# dynamic block.
variable "efs_volumes" {
  type = map(string)
  default = {
    "efs-vol" = "fs-12345678"
  }
}

resource "aws_ecs_task_definition" "pass" {
  family                = "service"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  dynamic "volume" {
    for_each = var.efs_volumes
    content {
      name = volume.key

      efs_volume_configuration {
        file_system_id     = volume.value
        transit_encryption = "ENABLED"
      }
    }
  }
}
