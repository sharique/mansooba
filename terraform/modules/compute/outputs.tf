output "public_ip" {
  description = "Elastic IP address of the EC2 instance."
  value       = aws_eip.app.public_ip
}

output "ssh_command" {
  description = "Command to SSH into the EC2 instance."
  value       = "ssh -i ~/.ssh/mansooba ec2-user@${aws_eip.app.public_ip}"
}
