resource "aws_instance" "nginx" {
ami                         = data.aws_ami.amazon_linux2.id
instance_type               = var.instance_type
subnet_id                   = aws_subnet.dmz_public.id
vpc_security_group_ids      = [aws_security_group.dmz_sg.id]
associate_public_ip_address = false

user_data = <<-EOF
#!/bin/bash
set -euxo pipefail
yum update -y
amazon-linux-extras enable nginx1
yum clean metadata
yum install -y nginx
systemctl enable nginx
cat >/usr/share/nginx/html/index.html <<'EOT'
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Nginx DMZ</title></head>
<body style="font-family:sans-serif;">
<h1>Nginx is running</h1>
<p>Deployed in DMZ with a public (white) IP.</p>
</body>
</html>
EOT
systemctl restart nginx
EOF

tags = merge(local.tags, {
Name = "${var.project_name}-nginx"
Role = "web"
})
}

resource "aws_eip" "nginx" {
domain = "vpc"

tags = merge(local.tags, {
Name = "${var.project_name}-eip"
})
}

resource "aws_eip_association" "nginx_assoc" {
allocation_id = aws_eip.nginx.id
instance_id   = aws_instance.nginx.id
}