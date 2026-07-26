from fastapi import APIRouter

from app.models.request import RagRequest
from app.models.response import RagResponse
from app.rag.rag_handler import rag_handler

router = APIRouter(
    prefix="/rag",
    tags=["RAG"]
)

@router.post(
    "/query",
    response_model=RagResponse
)
async def query_rag(request: RagRequest):

    return await rag_handler.ask(
        request.query,
        request.top_k
    )