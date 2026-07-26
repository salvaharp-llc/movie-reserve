from fastapi import FastAPI

from app.api.rag import router as rag_router
from app.config import settings

app = FastAPI(
    title=settings.APP_NAME,
    version=settings.VERSION
)

app.include_router(rag_router)


@app.get("/health")
async def health():
    return {
        "status": "ok"
    }