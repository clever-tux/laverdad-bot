#!/bin/bash
set -e

APP_NAME="event-bot"          # имя бинарника
SERVER_USER="antonkulikov"          # имя пользователя на сервере
SERVER_HOST="raspberrypi.local"   # IP или домен сервера
SERVER_PATH="/opt/laverdad-bot"  # путь на сервере
SERVICE_NAME="laverdad-bot"      # имя systemd-сервиса
DATABASE_USER="postgres"
DATABASE_PASS="postgres"
DATABASE_NAME="mafia_events_db"

echo "🫙 Creating DB"
ssh $SERVER_USER@$SERVER_HOST "sudo -u $DATABASE_USER psql -c 'create database $DATABASE_NAME; grant all privileges on database $DATABASE_NAME to $DATABASE_USER;'"

echo "🚂 Starting migrations"
# 123

echo "🔧 Creating system service"