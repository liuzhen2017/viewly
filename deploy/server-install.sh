#!/bin/bash
# Viewly server bootstrap — run once as ubuntu on the EC2 box.
set -e
sudo mkdir -p /opt/viewly /var/www
sudo mv /tmp/viewly-server /opt/viewly/server
sudo mv /tmp/config.prod.yaml /opt/viewly/config.yaml
sudo mv /tmp/static /opt/viewly/static
sudo rm -rf /var/www/h5 /var/www/admin
sudo mv /tmp/h5 /var/www/h5
sudo mv /tmp/adminui /var/www/admin
sudo chmod +x /opt/viewly/server

sudo tee /etc/systemd/system/viewly.service > /dev/null <<'EOF'
[Unit]
Description=Viewly API server
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/opt/viewly
ExecStart=/opt/viewly/server -config /opt/viewly/config.yaml
Restart=always
RestartSec=3
User=ubuntu

[Install]
WantedBy=multi-user.target
EOF

sudo apt-get update -qq
sudo apt-get install -y -qq nginx > /dev/null

sudo tee /etc/nginx/sites-available/viewly > /dev/null <<'EOF'
server {
    listen 80;
    server_name api.likeviewly.com;
    client_max_body_size 2m;
    location /static/ {
        alias /opt/viewly/static/;
        expires 7d;
    }
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
server {
    listen 80;
    server_name www.likeviewly.com likeviewly.com;
    root /var/www/h5;
    index index.html;
    location / {
        try_files $uri $uri/ /index.html;
    }
}
server {
    listen 80;
    server_name admin.likeviewly.com;
    root /var/www/admin;
    index index.html;
    location / {
        try_files $uri $uri/ /index.html;
    }
}
EOF
sudo ln -sf /etc/nginx/sites-available/viewly /etc/nginx/sites-enabled/viewly
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx

sudo systemctl daemon-reload
sudo systemctl enable --now viewly
sleep 2
curl -s http://127.0.0.1:8080/healthz && echo " API_OK"
echo INSTALL_DONE
