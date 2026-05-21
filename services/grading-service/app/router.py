import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.openapi.docs import get_swagger_ui_html
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.grading.worker import run_grading_consumer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("grading")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # ---- Startup ----
    logger.info("Grading service starting up...")
    logger.info("Connected to Kafka (brokers=%s)", settings.KAFKA_BROKERS)

    # Spawn the Kafka consumer as a background task
    consumer_task = asyncio.create_task(run_grading_consumer())
    logger.info("Grading Kafka consumer task started.")

    yield  # application runs

    # ---- Shutdown ----
    logger.info("Grading service shutting down...")
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        pass
    logger.info("Shutdown complete")


def create_app() -> FastAPI:
    app = FastAPI(
        title="Veritas Grading Service",
        description="Automated exam grading and evaluation service.",
        version="1.0.0",
        lifespan=lifespan,
        docs_url=None,
        openapi_url="/swagger/openapi.json"
    )

    @app.get("/swagger/index.html", include_in_schema=False)
    async def custom_swagger_ui_html():
        return get_swagger_ui_html(
            openapi_url="openapi.json",
            title=app.title + " - Swagger UI",
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
