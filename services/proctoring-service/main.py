from fastapi import FastAPI
import os

app = FastAPI()

@app.get("/health")
def health_check():
    return {"status": "OK"}

@app.get("/")
def read_root():
    service_name = os.getenv("SERVICE_NAME", os.getenv("HOSTNAME", "unknown-service"))
    return {"message": f"Hello from {service_name}"}

