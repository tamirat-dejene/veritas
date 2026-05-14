import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import RedirectResponse
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("grading")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # ---- Startup ----
    logger.info("Grading service starting up...")
    logger.info("Connected to Kafka (brokers=%s)", settings.KAFKA_BROKERS)
    
    yield  # application runs

    # ---- Shutdown ----
    logger.info("Grading service shutting down...")
    logger.info("Shutdown complete")


def create_app() -> FastAPI:
    app = FastAPI(
        title="Veritas Grading Service",
        description="Automated exam grading and evaluation service.",
        version="1.0.0",
        lifespan=lifespan,
        docs_url="/swagger/grading",
        openapi_url="/swagger/grading/openapi.json"
    )

    # Middleware
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Health check
    @app.get("/health", tags=["system"])
    async def health():
        return {"status": "OK", "service": "grading-service"}

    return app
