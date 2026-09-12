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
  region = var.aws_region
}

data "aws_caller_identity" "current" {}

output "aws_account_id" {
  value = data.aws_caller_identity.current.account_id
}

resource "aws_instance" "containerlab" {
  ami           = var.ami_id
  instance_type = var.instance_type

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

resource "aws_default_vpc" "containerlab" {
  tags = {
    Name = "Default VPC"
  }
}

resource "aws_subnet" "containerlab" {
  vpc_id                  = "vpc-0374055579a7c71fd"
  map_public_ip_on_launch = true

  tags = {
    Name = "Default Subnet 1b"
  }
}

resource "aws_internet_gateway" "containerlab" {
  vpc_id = "vpc-0374055579a7c71fd"

  tags = {
    Name = "Default Internet Gateway"
  }
}

resource "aws_route_table" "containerlab" {
  vpc_id = "vpc-0374055579a7c71fd"

  tags = {
    Name = "Default Rout Table"
  }
}
