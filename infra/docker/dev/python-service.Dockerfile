FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

RUN useradd --create-home --uid 10001 appuser \
    && mkdir -p /app/shared \
    && chown -R appuser:appuser /app

# The context is the service directory, so we just copy requirements.txt
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt uvicorn

USER appuser

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000", "--reload", "--reload-dir", "/app", "--reload-dir", "/app/shared", "--reload-include", "*.env"]
