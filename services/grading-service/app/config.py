from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    PY_PORT: int = 8000

    # Kafka
    KAFKA_BROKERS: str = "kafka:9092"

    # Hugging Face AI Space
    HF_EVALUATE_URL: str = (
        "https://your-huggingface-space-dummy-url.hf.space/evaluate"
    )
    HF_TIMEOUT_SECONDS: float = 120.0

    model_config = SettingsConfigDict(env_file=".env.dev", extra="ignore")


settings = Settings()

