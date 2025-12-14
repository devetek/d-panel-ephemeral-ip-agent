#!/bin/bash

# test curl loop to localhost:2220 and check status code
while true; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:2224/dashboard
  sleep 2
done