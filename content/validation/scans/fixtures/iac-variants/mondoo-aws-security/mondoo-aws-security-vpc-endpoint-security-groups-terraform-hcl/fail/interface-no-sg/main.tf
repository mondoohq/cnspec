resource "aws_vpc_endpoint" "example" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.us-east-1.ec2"
  vpc_endpoint_type = "Interface"
  subnet_ids        = [aws_subnet.main.id]
}
