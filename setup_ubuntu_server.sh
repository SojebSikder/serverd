#!/bin/bash

# Function to print messages
print_message() {
    echo -e "\e[1;32m$1\e[0m"
}

# Function to prompt for user input
prompt() {
    read -p "$1: " value
    echo "$value"
}

# prompt for yes or no
prompt_yes_no() {
    read -p "$1: " value
    if [[ "$value" == "y" || "$value" == "Y" ]]; then
        return 0
    fi
    return 1
}

git_clone_repo() {
    print_message "⚙️ Cloning repo..."
    read -p "Enter the repo name (e.g., user/repo): " REPO_NAME
    read -p "Enter the token: " TOKEN
    git clone https://$TOKEN@github.com/$REPO_NAME.git .
}

setup_firewall() {
    print_message "⚙️ Setting up user and firewall..."
    echo "$SUDO_PASSWORD" | sudo -S adduser "$USER_NAME"
    echo "$SUDO_PASSWORD" | sudo -S usermod -aG sudo "$USER_NAME"
    echo "$SUDO_PASSWORD" | sudo -S ufw allow OpenSSH
    echo "$SUDO_PASSWORD" | sudo -S ufw --force enable
}

setup_nginx() {
    print_message "⚙️ Installing Nginx..."
    echo "$SUDO_PASSWORD" | sudo -S apt update
    echo "$SUDO_PASSWORD" | sudo -S apt install -y nginx
    echo "$SUDO_PASSWORD" | sudo -S ufw allow 'Nginx HTTP'
    echo "$SUDO_PASSWORD" | sudo -S systemctl enable nginx
    echo "$SUDO_PASSWORD" | sudo -S systemctl start nginx
}

setup_nginx_server_block_for_nestjs() {
    print_message "⚙️ Setting up Nginx Server Block for NestJS..."
    echo "$SUDO_PASSWORD" | sudo -S mkdir -p /var/www/$PROJECT_NAME/backend
    echo "$SUDO_PASSWORD" | sudo -S chown -R $USER:$USER /var/www/$PROJECT_NAME/backend
    echo "$SUDO_PASSWORD" | sudo -S chmod -R 755 /var/www/$PROJECT_NAME/backend

    NGINX_CONF="/etc/nginx/sites-available/${PROJECT_NAME}_backend"
    echo "$SUDO_PASSWORD" | sudo -S bash -c "cat > $NGINX_CONF" <<EOL
server {
	listen 80;
	listen [::]:80;

	server_name $DOMAIN_NAME www.$DOMAIN_NAME;

	location / {
		proxy_pass http://localhost:4000;
		proxy_http_version 1.1;
		proxy_set_header Upgrade \$http_upgrade;
		proxy_set_header Connection 'upgrade';
		proxy_set_header Host \$host;
		proxy_cache_bypass \$http_upgrade;
	}

	location /public {
		root /var/www/$PROJECT_NAME/backend;
		add_header Access-Control-Allow-Origin *;
		add_header Access-Control-Allow-Methods 'GET, POST, OPTIONS';
		add_header Access-Control-Allow-Headers 'Content-Type, Authorization';
		try_files \$uri \$uri/ =404;
	}
}
EOL

    echo "$SUDO_PASSWORD" | sudo -S ln -s $NGINX_CONF /etc/nginx/sites-enabled/
    echo "$SUDO_PASSWORD" | sudo -S nginx -t
    echo "$SUDO_PASSWORD" | sudo -S systemctl restart nginx
}

setup_nginx_server_block_for_nextjs() {
    print_message "⚙️ Setting up Nginx Server Block for NextJS..."
    echo "$SUDO_PASSWORD" | sudo -S mkdir -p /var/www/$PROJECT_NAME/frontend
    echo "$SUDO_PASSWORD" | sudo -S chown -R $USER:$USER /var/www/$PROJECT_NAME/frontend
    echo "$SUDO_PASSWORD" | sudo -S chmod -R 755 /var/www/$PROJECT_NAME/frontend

    NGINX_CONF="/etc/nginx/sites-available/${PROJECT_NAME}_frontend"
    echo "$SUDO_PASSWORD" | sudo -S bash -c "cat > $NGINX_CONF" <<EOL
server {
	listen 80;
	listen [::]:80;

	server_name $DOMAIN_NAME www.$DOMAIN_NAME;

	location / {
		proxy_pass http://localhost:3000;
		proxy_http_version 1.1;
		proxy_set_header Upgrade \$http_upgrade;
		proxy_set_header Connection 'upgrade';
		proxy_set_header Host \$host;
		proxy_cache_bypass \$http_upgrade;
	}

	location /public {
		root /var/www/$PROJECT_NAME/frontend;
		add_header Access-Control-Allow-Origin *;
		add_header Access-Control-Allow-Methods 'GET, POST, OPTIONS';
		add_header Access-Control-Allow-Headers 'Content-Type, Authorization';
		try_files \$uri \$uri/ =404;
	}
}
EOL

    echo "$SUDO_PASSWORD" | sudo -S ln -s $NGINX_CONF /etc/nginx/sites-enabled/
    echo "$SUDO_PASSWORD" | sudo -S nginx -t
    echo "$SUDO_PASSWORD" | sudo -S systemctl restart nginx
}

setup_postgres() {
    print_message "⚙️ Installing PostgreSQL..."
    echo "$SUDO_PASSWORD" | sudo -S apt update
    echo "$SUDO_PASSWORD" | sudo -S apt install -y postgresql postgresql-contrib
    echo "$SUDO_PASSWORD" | sudo -S systemctl start postgresql.service
    echo "$SUDO_PASSWORD" | sudo -S -u postgres psql -c "ALTER USER postgres PASSWORD 'root'"
}

setup_nodejs() {
    print_message "⚙️ Installing Node.js & PM2..."
    curl -sL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    echo "$SUDO_PASSWORD" | sudo -S apt install -y nodejs
    echo "$SUDO_PASSWORD" | sudo -S npm install -g pm2 yarn
}

setup_nestjs_app() {
    print_message "⚙️ Running NestJS app..."
    cd /var/www/$PROJECT_NAME/backend
    git_clone_repo
    yarn install
    yarn build
    pm2 start dist/src/main.js --name "backend"
}

setup_nextjs_app() {
    print_message "⚙️ Running NextJS app..."
    cd /var/www/$PROJECT_NAME/frontend
    git_clone_repo
    yarn install
    yarn build
    pm2 start "yarn start" --name "frontend"
}

setup_ssl() {
    print_message "🔒 Setting up SSL with Let's Encrypt..."
    echo "$SUDO_PASSWORD" | sudo -S snap install core
    echo "$SUDO_PASSWORD" | sudo -S snap refresh core
    echo "$SUDO_PASSWORD" | sudo -S apt remove certbot -y
    echo "$SUDO_PASSWORD" | sudo -S snap install --classic certbot
    echo "$SUDO_PASSWORD" | sudo -S ln -s /snap/bin/certbot /usr/bin/certbot
    echo "$SUDO_PASSWORD" | sudo -S certbot --nginx -d "$DOMAIN_NAME" -d "www.$DOMAIN_NAME"
}

# Prompt for basic info
PROJECT_NAME=$(prompt "Enter the project name")
SERVER_IP=$(prompt "Enter the Server IP")
USER_NAME=$(prompt "Enter the username you want to create")
DOMAIN_NAME=$(prompt "Enter your domain name")
SUDO_PASSWORD=$(prompt "Enter sudo password")

prompt_yes_no "Do you need SSL? (y/n)"
SSL=$?

print_message "🔑 Starting server setup..."

setup_firewall
setup_nginx
setup_nginx_server_block_for_nestjs
setup_nginx_server_block_for_nextjs
setup_postgres
setup_nodejs
setup_nestjs_app
setup_nextjs_app

if [[ "$SSL" -eq 0 ]]; then
    setup_ssl
fi

print_message "✅ Setup Completed Successfully!"
