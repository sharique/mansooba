output "vpc_id" {
  description = "ID of the VPC."
  value       = aws_vpc.main.id
}

output "public_subnet_id" {
  description = "ID of the public subnet (place EC2 here)."
  value       = aws_subnet.public.id
}

output "private_subnet_ids" {
  description = "IDs of the two private subnets (place RDS subnet group here)."
  value       = [aws_subnet.private_a.id, aws_subnet.private_b.id]
}
