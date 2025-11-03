resource "aws_security_group" "dmz_sg" {
name        = "${var.project_name}-dmz-sg"
description = "DMZ security group allowing HTTP/HTTPS and SSH"
vpc_id      = aws_vpc.dmz.id

ingress {
description = "HTTP"
protocol    = "tcp"
from_port   = 80
to_port     = 80
cidr_blocks = ["0.0.0.0/0"]
}

ingress {
description = "HTTPS"
protocol    = "tcp"
from_port   = 443
to_port     = 443
cidr_blocks = ["0.0.0.0/0"]
}

ingress {
description = "SSH"
protocol    = "tcp"
from_port   = 22
to_port     = 22
cidr_blocks = [var.allowed_ssh_cidr]
}

egress {
description = "All egress"
protocol    = "-1"
from_port   = 0
to_port     = 0
cidr_blocks = ["0.0.0.0/0"]
}

tags = merge(local.tags, {
Name = "${var.project_name}-dmz-sg"
})
}