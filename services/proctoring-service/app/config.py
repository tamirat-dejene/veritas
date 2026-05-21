import os
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    PY_PORT: int = 8000

    # Database
    PG_VERITAS_HOST: str = "postgres"
    PG_VERITAS_PORT: int = 5432
    PG_VERITAS_USER: str = "postgres"
    PG_VERITAS_PASSWORD: str = "postgres"
    PG_VERITAS_CORE_DB: str = "veritas_core"

    # Kafka
    KAFKA_BROKERS: str = "kafka:9092"

    # Redis
    REDIS_HOST: str = "redis"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: str = ""
    REDIS_DB: int = 0

    # Auth
    JWT_SECRET: str = "supersecretkey123"

    # Internal service URLs
    CANDIDATE_SERVICE_URL: str = "http://candidate-service:8080"

    # Face detection provider
    FACE_API_URL: str = "https://deepface-api-XXXXXXXXXX.us-central1.run.app"
    FACE_API_MODEL: str = "Facenet512"
    FACE_API_DETECTOR: str = "retinaface"

    model_config = SettingsConfigDict(extra="ignore")

    @property
    def DATABASE_URL(self) -> str:
        db_name = os.getenv("POSTGRES_PROCTORING_DB") or os.getenv("PG_VERITAS_CORE_DB") or self.PG_VERITAS_CORE_DB
        return (
            f"postgresql://{self.PG_VERITAS_USER}:{self.PG_VERITAS_PASSWORD}"
            f"@{self.PG_VERITAS_HOST}:{self.PG_VERITAS_PORT}/{db_name}"
        )


settings = Settings()
