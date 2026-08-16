resource "aws_launch_template" "noncompliant" {
  name          = "example"
  image_id      = "ami-12345678"
  instance_type = "t3.micro"
  user_data     = "DB_PASSWORD=Sup3rSecretP@ss123"
}
