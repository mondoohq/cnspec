# Non-compliant: MQ broker is publicly accessible.
resource "aws_mq_broker" "fail_example" {
  broker_name        = "fail-example"
  engine_type        = "ActiveMQ"
  publicly_accessible = true
}
