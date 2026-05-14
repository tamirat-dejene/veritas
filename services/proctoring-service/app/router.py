"""
FastAPI application factory + lifespan.

Dependency wiring order:
  config → database pool → repositories → usecases → handlers

The FaceDetector is injected here — swap DeepFaceDetector for
RekognitionDetector in this file only, no other changes needed.
"""
import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.openapi.docs import get_swagger_ui_html
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.database import create_pool
from app.middleware.context import IdentityMiddleware

# Repositories
from app.repository.event_repository import EventRepository
from app.repository.score_repository import ScoreRepository

# Infrastructure
# from app.infrastructure.face.deepface_detector import DeepFaceDetector
from app.infrastructure.face.deepface_dev_detector import DeepFaceDevDetector
from app.infrastructure.client.candidate_client import CandidateServiceClient
from app.infrastructure.kafka.producer import KafkaProducer
from app.infrastructure.kafka.consumer import run_consumer

# Usecases
from app.usecase.event_usecase import EventUseCase
from app.usecase.face_usecase import FaceUseCase

# Handlers
from app.handler import face_handler, event_handler, health_handler

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("proctoring")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # ---- Startup ----
    logger.info("Proctoring service starting up...")

    # 1. Database
    pool = await create_pool()
    app.state.pool = pool
    logger.info("Connected to PostgreSQL")

    # 2. Kafka producer
    producer = KafkaProducer()
    await producer.start()
    app.state.producer = producer
    logger.info("Kafka producer started")

    # 3. Repositories
    event_repo = EventRepository(pool)
    score_repo = ScoreRepository(pool)

    # 4. Infrastructure — face detector (auto-select based on config)
    if settings.DEEPFACE_DEV_API_KEY:
        detector = DeepFaceDevDetector(
            api_key=settings.DEEPFACE_DEV_API_KEY,
            model_name=settings.DEEPFACE_DEV_MODEL,
            detector_backend=settings.DEEPFACE_DEV_DETECTOR,
        )
        logger.info("Face detector: deepface.dev managed API (model=%s)", settings.DEEPFACE_DEV_MODEL)
    else:
        logger.warning("DEEPFACE_DEV_API_KEY is empty. Falling back to local DeepFace detector (requires local dependencies).")
        # Lazy import — only load when actually needed to avoid ImportError
        from app.infrastructure.face.deepface_detector import DeepFaceDetector
        detector = DeepFaceDetector()
        logger.info("Face detector: self-hosted DeepFace (local)")
    candidate_client = CandidateServiceClient(settings.CANDIDATE_SERVICE_URL)

    # 5. Usecases
    event_uc = EventUseCase(event_repo, score_repo, producer)
    face_uc = FaceUseCase(detector, candidate_client, event_uc, producer)

    app.state.event_uc = event_uc
    app.state.face_uc = face_uc

    # 6. Kafka consumer (background task)
    consumer_task = asyncio.create_task(run_consumer(event_uc, producer))
    logger.info("Kafka consumer task started")

    logger.info("Proctoring service ready on port %d", settings.PY_PORT)

    yield  # application runs

    # ---- Shutdown ----
    logger.info("Proctoring service shutting down...")
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        pass
    await producer.stop()
    await pool.close()
    logger.info("Shutdown complete")


def create_app() -> FastAPI:
    app = FastAPI(
        title="Veritas Proctoring Service",
        description=(
            "AI-based candidate activity monitoring, face identity verification, "
            "behavioral event logging, and cheating probability scoring."
        ),
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
    app.add_middleware(IdentityMiddleware)
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Routers
    app.include_router(health_handler.router, tags=["system"])
    app.include_router(face_handler.router, tags=["face"])
    app.include_router(event_handler.router, tags=["proctoring"])

    return app
