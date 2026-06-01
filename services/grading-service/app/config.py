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

    # Hugging Face AI Space
    HF_EVALUATE_URL: str = ("https://cheifo-YYYYYYYYYYYYY.hf.space/evaluate")
    HF_TIMEOUT_SECONDS: float = 120.0
    HF_TOKEN: str | None = None

    GRADING_SECRET_KEY: str = "supersecretkey123"

    # Candidate service (internal service-to-service calls)
    CANDIDATE_SERVICE_URL: str = "http://candidate-service:8080"
    CANDIDATE_SERVICE_TIMEOUT_SECONDS: float = 30.0

    # Enterprise service (internal service-to-service calls)
    ENTERPRISE_SERVICE_URL: str = "http://enterprise-service:8080"

    model_config = SettingsConfigDict(env_file=".env.dev", extra="ignore")

    @property
    def DATABASE_URL(self) -> str:
        db_name = os.getenv("POSTGRES_GRADING_DB") or os.getenv("PG_VERITAS_CORE_DB") or self.PG_VERITAS_CORE_DB
        return (
            f"postgresql://{self.PG_VERITAS_USER}:{self.PG_VERITAS_PASSWORD}"
            f"@{self.PG_VERITAS_HOST}:{self.PG_VERITAS_PORT}/{db_name}"
        )

settings = Settings()