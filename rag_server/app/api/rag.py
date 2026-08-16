import uuid

from fastapi import APIRouter, status, Response

from app.models.request import AddRequest, RagRequest, UpdateRequest
from app.models.response import RagResponse
from app.rag.rag_service import rag_service

router = APIRouter(
    prefix="/rag",
    tags=["RAG"]
)

@router.post(
    "/query",
    response_model=RagResponse
)
async def query_rag(request: RagRequest):

    return await rag_service.ask(
        request.query,
        request.top_k,
        request.rerank,
        request.enhance
    )

@router.post(
    "/movies",
    status_code=status.HTTP_201_CREATED,
)
async def add_document(request: AddRequest, response: Response):

    err = await rag_service.add(
        id=request.id,
        title=request.title,
        description=request.description
    )
    if err:
        response.status_code = status.HTTP_400_BAD_REQUEST
        return {"error": str(err)}

@router.put(
    "/movies/{id}",
    status_code=status.HTTP_204_NO_CONTENT,
)
async def update_document(id: uuid.UUID, request: UpdateRequest, response: Response):

    err = await rag_service.update(
        id=id,
        title=request.title,
        description=request.description
    )
    if err:
        response.status_code = status.HTTP_400_BAD_REQUEST
        return {"error": str(err)}

@router.delete(
    "/movies/{id}",
    status_code=status.HTTP_204_NO_CONTENT,
)
async def delete_document(id: uuid.UUID, response: Response):

    err = await rag_service.delete(
        id=id
    )
    if err:
        response.status_code = status.HTTP_400_BAD_REQUEST
        return {"error": str(err)}