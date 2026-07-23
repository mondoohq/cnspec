# Non-compliant: InfluxDB instance is publicly accessible.
resource "aws_timestreaminfluxdb_db_instance" "public" {
  name                   = "prod-influx"
  allocated_storage      = 20
  bucket                 = "prod-metrics"
  db_instance_type       = "db.influx.medium"
  password               = "supersecretpassword"
  username               = "admin"
  organization           = "acme"
  vpc_subnet_ids         = ["subnet-0123456789abcdef0"]
  vpc_security_group_ids = ["sg-0123456789abcdef0"]
  publicly_accessible    = true
}
