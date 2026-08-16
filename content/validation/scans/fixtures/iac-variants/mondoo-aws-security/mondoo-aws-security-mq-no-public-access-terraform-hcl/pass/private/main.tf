# Compliant: MQ broker is not publicly accessible.
resource "aws_mq_broker" "pass_example" {
  broker_name        = "pass-example"
  engine_type        = "ActiveMQ"
  publicly_accessible = false
}
