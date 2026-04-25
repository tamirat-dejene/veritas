#!/bin/bash

COMPOSE_FILE="/home/tamirat-dejene/Documents/coursework/final_project/veritas/docker-compose.yml"

docker compose -f "$COMPOSE_FILE" logs -f --tail=100 -t \
  api-gateway auth-service candidate-service enterprise-service exam-service notification-service \
  face-verification-service grading-service grading-service payment-service proctoring-service