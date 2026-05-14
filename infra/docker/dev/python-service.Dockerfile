FROM python:3.12-slim

ARG SERVICE_NAME=unknown
LABEL service=$SERVICE_NAME

# Prevent Python from writing .pyc files and enable unbuffered logging
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    VIRTUAL_ENV=/app/.venv \
    PATH="/app/.venv/bin:$PATH"

WORKDIR /app

# Install system dependencies if needed
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Setup virtual environment and user
RUN python -m venv $VIRTUAL_ENV \
    && pip install --no-cache-dir --upgrade pip setuptools wheel \
    && useradd --create-home --uid 10001 appuser \
    && chown -R appuser:appuser /app $VIRTUAL_ENV

# Copy and install dependencies as root to use cache efficiently
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

USER appuser

EXPOSE 8000

# Start uvicorn with hot-reload enabled for development
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--reload", "--reload-dir", ".", "--reload-include", "*.py", "--reload-include", "*.env*"]
