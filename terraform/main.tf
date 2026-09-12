terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  required_version = ">= 1.16.0"
}

provider "aws" {
  region = "ap-south-1"
}

data "aws_caller_identity" "current" {}

output "aws_account_id" {
  value = data.aws_caller_identity.current.account_id
}

resource "aws_instance" "containerlab" {
  ami           = "ami-08188a5a4dfdbd573"
  instance_type = "t3.micro"

  tags = {
    Name = "containerlab-server"
  }
}

resource "aws_security_group" "containerlab" {
  name        = "launch-wizard-13"
  description = "launch-wizard-13 created 2026-09-10T07:09:16.912Z"
  vpc_id      = "vpc-0374055579a7c71fd"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["106.206.196.202/32"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
