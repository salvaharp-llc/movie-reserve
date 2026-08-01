from fastapi import FastAPI, Depends

from app.api.rag import router as rag_router
from app.config import settings
from app.security import verify_api_key

app = FastAPI(
    title=settings.APP_NAME,
    version=settings.VERSION,
)

app.include_router(
    rag_router,
    dependencies=[Depends(verify_api_key)],
)


@app.get("/health")
async def health():
    return {"status": "ok"}