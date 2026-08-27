R=ap-east-1
B=viewly-videos-386344085984

echo "== 1/6 import key pair =="
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKJHOukHtJQOrXm2RUg6Tg0qzePvX6MysjXVeVjhVLxx viewly-hk-deploy" > /tmp/k.pub
aws ec2 import-key-pair --region $R --key-name viewly-hk --public-key-material fileb:///tmp/k.pub --query KeyName --output text

echo "== 2/6 security group =="
SGID=$(aws ec2 describe-security-groups --region $R --group-names viewly-sg --query 'SecurityGroups[0].GroupId' --output text 2>/dev/null || true)
if [ "$SGID" = "None" ] || [ -z "$SGID" ]; then SGID=$(aws ec2 create-security-group --region $R --group-name viewly-sg --description "viewly app" --query GroupId --output text); fi
aws ec2 authorize-security-group-ingress --region $R --group-id $SGID --protocol tcp --port 22 --cidr 155.254.122.132/32
aws ec2 authorize-security-group-ingress --region $R --group-id $SGID --protocol tcp --port 80 --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --region $R --group-id $SGID --protocol tcp --port 443 --cidr 0.0.0.0/0
echo "SGID=$SGID"

echo "== 3/6 IAM role =="
cat > /tmp/trust.json <<'EOF'
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}
EOF
cat > /tmp/pol.json <<EOF
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:PutObject"],"Resource":"arn:aws:s3:::$B/*"}]}
EOF
aws iam create-role --role-name viewly-ec2-s3put --assume-role-policy-document file:///tmp/trust.json || true
aws iam put-role-policy --role-name viewly-ec2-s3put --policy-name s3put --policy-document file:///tmp/pol.json
aws iam create-instance-profile --instance-profile-name viewly-ec2-s3put || true
aws iam add-role-to-instance-profile --instance-profile-name viewly-ec2-s3put --role-name viewly-ec2-s3put || true

echo "== 4/6 S3 bucket =="
aws s3api create-bucket --bucket $B --region $R --create-bucket-configuration LocationConstraint=$R
aws s3api put-public-access-block --bucket $B --public-access-configuration BlockPublicAcls=false,IgnorePublicAcls=false,BlockPublicPolicy=false,RestrictPublicBuckets=false
cat > /tmp/bkt.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowCFOnly",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::viewly-videos-386344085984/*",
      "Condition": {
        "IpAddress": {
          "aws:SourceIp": [
            "173.245.48.0/20","103.21.244.0/22","103.22.200.0/22","103.31.4.0/22",
            "141.101.64.0/18","108.162.192.0/18","190.93.240.0/20","188.114.96.0/20",
            "197.234.240.0/22","198.41.128.0/17","162.158.0.0/15","104.16.0.0/13",
            "104.24.0.0/14","172.64.0.0/13","131.0.72.0/22",
            "2400:cb00::/32","2606:4700::/32","2803:f800::/32","2405:b500::/32",
            "2405:8100::/32","2a06:98c0::/29","2c0f:f248::/32"
          ]
        }
      }
    }
  ]
}
EOF
aws s3api put-bucket-policy --bucket $B --policy file:///tmp/bkt.json

echo "== 5/6 launch EC2 =="
AMI=$(aws ssm get-parameters --names /aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id --query 'Parameters[0].Value' --output text)
echo "AMI=$AMI"
sleep 10
IID=$(aws ec2 run-instances --region $R --image-id $AMI --instance-type t3.small --key-name viewly-hk --security-group-ids $SGID --iam-instance-profile Name=viewly-ec2-s3put --credit-specification CpuCredits=unlimited --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":30,"VolumeType":"gp3"}}]' --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=viewly-app}]' --metadata-options HttpTokens=required --query 'Instances[0].InstanceId' --output text)
echo "IID=$IID"

echo "== 6/6 elastic IP =="
EIPA=$(aws ec2 allocate-address --region $R --query AllocationId --output text)
sleep 8
aws ec2 associate-address --region $R --instance-id $IID --allocation-id $EIPA
aws ec2 describe-addresses --region $R --allocation-ids $EIPA --query 'Addresses[0].PublicIp' --output text
echo "ALL_DONE SGID=$SGID IID=$IID"
