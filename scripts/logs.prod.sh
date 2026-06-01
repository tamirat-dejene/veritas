#!/bin/bash

COMPOSE_FILE="veritas/docker-compose.prod.yml"

docker-compose -f "$COMPOSE_FILE" logs -f --tail=100 -t \
  api-gateway auth-service candidate-service enterprise-service exam-service notification-service \
  grading-service payment-service proctoring-service