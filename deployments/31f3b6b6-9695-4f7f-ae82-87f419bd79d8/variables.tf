variable "region" {
description = "AWS region"
type        = string
default     = "us-east-1"
}

variable "project_name" {
description = "Project name for tagging"
type        = string
default     = "nginx-dmz"
}

variable "instance_type" {
description = "EC2 instance type"
type        = string
default     = "t3.micro"
}

variable "allowed_ssh_cidr" {
description = "CIDR block allowed to SSH into the instance"
type        = string
default     = "0.0.0.0/0"
}

variable "vpc_cidr" {
description = "CIDR block for the VPC"
type        = string
default     = "10.50.0.0/16"
}

variable "dmz_subnet_cidr" {
description = "CIDR block for the DMZ public subnet"
type        = string
default     = "10.50.10.0/24"
}