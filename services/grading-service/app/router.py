import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.openapi.docs import get_swagger_ui_html
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.database import create_pool
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

    # 2. Dependency Injection
    grading_repo = GradingRepository(pool)
    grading_uc = GradingUseCase(grading_repo)
    app.state.grading_uc = grading_uc

    # 3. Spawn the Kafka consumer as a background task, passing the DB pool
    consumer_task = asyncio.create_task(run_grading_consumer(pool))
    logger.info("Grading Kafka consumer task started.")

    yield  # application runs

    # ---- Shutdown ----
    logger.info("Grading service shutting down...")
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        pass
    
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
    app.add_middleware(IdentityMiddleware)

    # Routers
    app.include_router(grading_handler.router)

    # Health check
    @app.get("/health", tags=["system"])
    async def health():
        return {"status": "OK", "service": "grading-service"}

    return app
