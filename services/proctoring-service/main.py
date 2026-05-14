"""
Proctoring Service entrypoint.
"""
import uvicorn
from app.config import settings
from app.router import create_app

app = create_app()

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=settings.PY_PORT,
        reload=False,
    )
