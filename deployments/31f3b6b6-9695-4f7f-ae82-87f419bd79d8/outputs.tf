output "vpc_id" {
value       = aws_vpc.dmz.id
description = "VPC ID"
}

output "dmz_subnet_id" {
value       = aws_subnet.dmz_public.id
description = "DMZ public subnet ID"
}

output "instance_id" {
value       = aws_instance.nginx.id
description = "EC2 instance ID"
}

output "public_ip" {
value       = aws_eip.nginx.public_ip
description = "Public EIP of the Nginx instance"
}

output "nginx_url" {
value       = "http://${aws_eip.nginx.public_ip}"
description = "HTTP URL to access Nginx"
}