terraform {
required_version = ">= 1.4.0"
required_providers {
aws = {
source  = "hashicorp/aws"
version = "~> 5.0"
}
}
}

provider "aws" {
region = var.region
}

data "aws_availability_zones" "available" {
state = "available"
}

data "aws_ami" "amazon_linux2" {
most_recent = true
owners      = ["amazon"]

filter {
name   = "name"
values = ["amzn2-ami-hvm-*-x86_64-gp2"]
}

filter {
name   = "virtualization-type"
values = ["hvm"]
}

filter {
name   = "root-device-type"
values = ["ebs"]
}
}

locals {
az = data.aws_availability_zones.available.names[0]
tags = {
Project = var.project_name
Managed = "terraform"
}
}