output "instance_profile_name" {
  description = "Name of the IAM instance profile to attach to EC2."
  value       = aws_iam_instance_profile.ec2.name
}
