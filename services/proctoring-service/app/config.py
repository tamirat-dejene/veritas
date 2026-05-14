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

    # Auth
    JWT_SECRET: str = "supersecretkey123"

    # Internal service URLs
    CANDIDATE_SERVICE_URL: str = "http://candidate-service:8080"

    # Face detection provider — set to use deepface.dev managed API instead of local DeepFace.
    # Leave empty to use self-hosted DeepFace (default).
    # Sign up at https://deepface.dev/signup to get a key.
    DEEPFACE_DEV_API_KEY: str = ""
    DEEPFACE_DEV_MODEL: str = "Facenet512"       # Facenet, Facenet512, Dlib, OpenFace, SFace
    DEEPFACE_DEV_DETECTOR: str = "retinaface"    # retinaface, opencv, mtcnn, yolov8

    model_config = SettingsConfigDict(extra="ignore")

    @property
    def DATABASE_URL(self) -> str:
        return (
            f"postgresql://{self.PG_VERITAS_USER}:{self.PG_VERITAS_PASSWORD}"
            f"@{self.PG_VERITAS_HOST}:{self.PG_VERITAS_PORT}/{self.PG_VERITAS_CORE_DB}"
        )


settings = Settings()
