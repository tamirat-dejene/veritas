"""
asyncpg connection pool — created once during FastAPI lifespan,
shared across all requests via app.state.pool.
"""
import asyncpg
from app.config import settings


async def create_pool() -> asyncpg.Pool:
    return await asyncpg.create_pool(
        dsn=settings.DATABASE_URL,
        min_size=2,
        max_size=10,
        command_timeout=30,
    )
