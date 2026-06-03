import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.openapi.docs import get_swagger_ui_html
from fastapi.middleware.cors import CORSMiddleware
from fastapi.exceptions import RequestValidationError
from fastapi.exception_handlers import request_validation_exception_handler

from app.config import settings
from app.database import create_pool
from app.grading.candidate_client import CandidateServiceClient
from app.grading.enterprise_client import EnterpriseServiceClient
from app.grading.producer import KafkaProducer
from app.grading.worker import run_grading_consumer
from app.middleware.context import IdentityMiddleware
from app.repository.grading_repository import GradingRepository
from app.usecase.grading_usecase import GradingUseCase
from app.handler import grading_handler

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("grading")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # ---- Startup ----
    logger.info("Grading service starting up...")

    # 1. Database Connection Pool Setup
    pool = await create_pool()
    app.state.pool = pool
    logger.info("Connected to PostgreSQL")

    # 2. Start service clients & Kafka producer
    candidate_client = CandidateServiceClient()
    await candidate_client.start()
    app.state.candidate_client = candidate_client
    logger.info(
        "CandidateServiceClient started — base_url=%s",
        settings.CANDIDATE_SERVICE_URL,
    )

    enterprise_client = EnterpriseServiceClient()
    await enterprise_client.start()
    app.state.enterprise_client = enterprise_client
    logger.info(
        "EnterpriseServiceClient started — base_url=%s",
        settings.ENTERPRISE_SERVICE_URL,
    )

    kafka_producer = KafkaProducer()
    await kafka_producer.start()
    app.state.kafka_producer = kafka_producer
    logger.info("Kafka producer started")

    # 3. Dependency Injection
    grading_repo = GradingRepository(pool)
    grading_uc = GradingUseCase(grading_repo, candidate_client, enterprise_client)
    app.state.grading_uc = grading_uc

    # 4. Spawn the Kafka consumer as a background task
    consumer_task = asyncio.create_task(
        run_grading_consumer(pool, candidate_client, kafka_producer)
    )
    logger.info("Grading Kafka consumer task started.")

    yield  # application runs

    # ---- Shutdown ----
    logger.info("Grading service shutting down...")
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        pass

    await kafka_producer.stop()
    logger.info("Kafka producer stopped.")

    await candidate_client.stop()
    logger.info("CandidateServiceClient stopped.")

    await enterprise_client.stop()
    logger.info("EnterpriseServiceClient stopped.")

    await pool.close()
    logger.info("Shutdown complete")


def create_app() -> FastAPI:
    app = FastAPI(
        title="Veritas Grading Service",
        description="Automated exam grading, manual override, and audit trail validation service.",
        version="1.0.0",
        lifespan=lifespan,
        docs_url=None,
        openapi_url="/swagger/openapi.json"
    )

    @app.exception_handler(RequestValidationError)
    async def validation_exception_handler(request, exc: RequestValidationError):
        logger.error(
            "Request validation failed for %s %s: %s",
            request.method,
            request.url.path,
            exc.errors(),
        )
        return await request_validation_exception_handler(request, exc)

    @app.get("/swagger/index.html", include_in_schema=False)
    async def custom_swagger_ui_html():
        return get_swagger_ui_html(
            openapi_url="openapi.json",
            title=app.title + " - Swagger UI",
        )

    app.add_middleware(IdentityMiddleware)

    # Routers
    app.include_router(grading_handler.router)

    # Health check
    @app.get("/health", tags=["system"])
    async def health():
        return {"status": "OK", "service": "grading-service"}

    return app
