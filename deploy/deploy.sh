#!/bin/bash
set -e

APP_NAME="event-bot"          # имя бинарника
SERVER_USER="antonkulikov"          # имя пользователя на сервере
SERVER_HOST="raspberrypi.local"   # IP или домен сервера
SERVER_PATH="/opt/laverdad-bot"  # путь на сервере
SERVICE_NAME="laverdad-bot"      # имя systemd-сервиса

echo "🚀 Building Go binary..."
GOOS=linux GOARCH=arm64 go build -o $APP_NAME .

echo "🛑 Stoping service on server..."
ssh $SERVER_USER@$SERVER_HOST "sudo systemctl stop $SERVICE_NAME"

echo "📦 Copying files to server..."
scp $APP_NAME $SERVER_USER@$SERVER_HOST:$SERVER_PATH/
scp .env $SERVER_USER@$SERVER_HOST:$SERVER_PATH/

echo "🔄 Restarting service on server..."
ssh $SERVER_USER@$SERVER_HOST "sudo systemctl restart $SERVICE_NAME && sudo systemctl status $SERVICE_NAME --no-pager"

echo "✅ Deploy completed!"
