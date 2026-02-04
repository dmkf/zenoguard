#!/bin/bash
# Deploy script for zenoGuard agent
# Password: nash123347629

set -e

echo "Deploying zenoGuard agent to 10.0.0.200..."

# Upload agent
echo "Uploading agent binary..."
scp zenoguard-agent-linux-amd64 xiong@10.0.0.200:/tmp/zenoguard-agent-new

# Restart service on remote server
echo "Restarting agent service..."
ssh xiong@10.0.0.200 << 'ENDSSH'
sudo cp /tmp/zenoguard-agent-new /usr/local/bin/zenoguard-agent
sudo chmod +x /usr/local/bin/zenoguard-agent
sudo systemctl restart zenoguard-agent
sudo systemctl status zenoguard-agent --no-pager
ENDSSH

echo "✓ Deployment complete!"
