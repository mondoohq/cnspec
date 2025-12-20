terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.74"
    }
  }
}

provider "aws" {
  # Configuration options
}