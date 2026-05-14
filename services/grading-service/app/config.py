from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    PY_PORT: int = 8000

    # Kafka
    KAFKA_BROKERS: str = "kafka:9092"

    model_config = SettingsConfigDict(env_file=".env.dev", extra="ignore")


settings = Settings()
