resource "aws_ebs_volume" "fail_example" {
  availability_zone = "us-east-1"
  size              = 40
  tags = {
    Name = "Not Encrypted"
  }
  encrypted = false
}

resource "aws_instance" "fail_example" {
  ami           = "ami-0279c3b3186e54acd"
  instance_type = "t2.micro"
}

resource "aws_instance" "fail_example" {
  ami           = "ami-0279c3b3186e54acd"
  instance_type = "t2.micro"
  # we explicitly do not use the sample account
  user_data = <<EOF
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7AAAAAAA
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/A1AAAAA/bPxRfiCYAAAAAAAKEY
export AWS_DEFAULT_REGION=us-east-1
EOF
}