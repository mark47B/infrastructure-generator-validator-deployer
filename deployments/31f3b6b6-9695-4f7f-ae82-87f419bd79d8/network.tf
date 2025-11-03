resource "aws_vpc" "dmz" {
cidr_block           = var.vpc_cidr
enable_dns_hostnames = true
enable_dns_support   = true

tags = merge(local.tags, {
Name = "${var.project_name}-vpc"
Zone = "dmz"
})
}

resource "aws_internet_gateway" "this" {
vpc_id = aws_vpc.dmz.id

tags = merge(local.tags, {
Name = "${var.project_name}-igw"
})
}

resource "aws_subnet" "dmz_public" {
vpc_id                  = aws_vpc.dmz.id
cidr_block              = var.dmz_subnet_cidr
availability_zone       = local.az
map_public_ip_on_launch = false

tags = merge(local.tags, {
Name = "${var.project_name}-dmz-public"
Tier = "dmz"
Public = "true"
})
}

resource "aws_route_table" "public" {
vpc_id = aws_vpc.dmz.id

route {
cidr_block = "0.0.0.0/0"
gateway_id = aws_internet_gateway.this.id
}

tags = merge(local.tags, {
Name = "${var.project_name}-public-rt"
})
}

resource "aws_route_table_association" "dmz_public_assoc" {
subnet_id      = aws_subnet.dmz_public.id
route_table_id = aws_route_table.public.id
}