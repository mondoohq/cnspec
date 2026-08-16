# container_definitions is a JSON string; the only idiomatic form is jsonencode([...]).
# NOTE: provider bug — jsonencode([...]) evaluates to [] so this check is currently
# unenforceable on terraform-hcl (see fail/IMPOSSIBLE.md). Provider PR tracks the fix.
resource "aws_ecs_task_definition" "pass" {
  family                = "app"
  container_definitions = jsonencode([{ name = "app", image = "nginx", user = "1000" }])
}
