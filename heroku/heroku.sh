#!/bin/bash

### Parameters
# Jitsu port
NGINX_PORT_VALUE=$PORT
if [[ -z "$NGINX_PORT_VALUE" ]]; then
  NGINX_PORT_VALUE=8000
fi

# Jitsu Server admin token
if [[ -z "$SERVER_ADMIN_TOKEN" ]]; then
  export SERVER_ADMIN_TOKEN=$(date +%s|sha256sum|base64|head -c 32)
fi

# Jitsu Configurator admin token
if [[ -z "$CONFIGURATOR_ADMIN_TOKEN" ]]; then
  export CONFIGURATOR_ADMIN_TOKEN=$(date +%s|sha256sum|base64|head -c 32)
fi

# Jitsu UI authorization access secret
if [[ -z "$UI_AUTH_ACCESS_SECRET" ]]; then
  export UI_AUTH_ACCESS_SECRET=$(date +%s|sha256sum|base64|head -c 32)
fi

# Jitsu UI authorization refresh secret
if [[ -z "$UI_AUTH_REFRESH_SECRET" ]]; then
  export UI_AUTH_REFRESH_SECRET=$(date +%s|sha256sum|base64|head -c 32)
fi

### Start services
# Start Jitsu Configurator process
nohup /home/configurator/app/configurator -cfg=/home/configurator/data/config/configurator.yaml -cr=true -dhid=heroku &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start Jitsu Configurator: $status"
  exit $status
fi

sleep 1

# Start Jitsu Server process
nohup /home/eventnative/app/eventnative -cfg=/home/eventnative/data/config/eventnative.yaml -cr=true -dhid=heroku &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start Jitsu Server : $status"
  exit $status
fi

sleep 1

# Start Nginx process
sed "s/NGINX_PORT/$NGINX_PORT_VALUE/g" /etc/nginx/nginx.conf > /etc/nginx/nginx_replaced.conf && \
mv /etc/nginx/nginx_replaced.conf /etc/nginx/nginx.conf && \
nohup nginx -g 'daemon off;' &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start Nginx : $status"
  exit $status
fi

sleep 1


# Naive check runs checks once a minute to see if either of the processes exited.
# This illustrates part of the heavy lifting you need to do if you want to run
# more than one service in a container. The container exits with an error
# if it detects that either of the processes has exited.
# Otherwise it loops forever, waking up every 60 seconds

while sleep 5; do
  ps aux |grep configurator |grep -q -v grep
  PROCESS_CONFIGURATOR=$?
  ps aux |grep eventnative |grep -q -v grep
  PROCESS_SERVER=$?
  ps aux |grep nginx |grep -q -v grep
  PROCESS_NGINX=$?
  # If the greps above find anything, they exit with 0 status
  # If they are not both 0, then something is wrong
  if [ $PROCESS_CONFIGURATOR -ne 0 ]; then
    echo "Jitsu Configurator has already exited."
    exit 1
  fi
  if [ $PROCESS_SERVER -ne 0 ]; then
    echo "Jitsu Server has already exited."
    exit 1
  fi
  if [ $PROCESS_NGINX -ne 0 ]; then
    echo "Nginx has already exited."
    exit 1
  fi
done