import asyncio
from binascii import Error
from typing import Optional
import uuid

from app.models.response import RagResponse, Source
from app.database.movies import Querier
from app.config import settings
import sqlalchemy
from app.rag.lib.augmented_generation import (
    HybridSearch, 
    format_search_results, 
    build_prompt,
    async_client,
    GEN_MODEL,
    DEFAULT_SEARCH_LIMIT,
)

class RagService:

    def __init__(self):
        db = sqlalchemy.create_engine(settings.DB_URL).connect()
        queries = Querier(db)
        docs = [
            {
                "id": movie.id, 
                "title": movie.title, 
                "description": movie.description if movie.description else ""
            } 
            for movie in queries.load_movies()
        ]
        self.hybrid_search = HybridSearch(docs)
        self._lock = asyncio.Lock()

    async def ask(self, query: str, top_k: Optional[int]) -> RagResponse:
        if top_k is None:
            top_k = DEFAULT_SEARCH_LIMIT

        async with self._lock:
            results = await asyncio.to_thread(
                self.hybrid_search.rrf_search, 
                query, 
                limit=top_k
            )

        if not results:
            return RagResponse(
                answer="",
                sources=[]
            )

        docs = format_search_results(results)

        prompt = build_prompt(query, docs)
    
        response = await async_client.chat.completions.create(
            model=GEN_MODEL,
            messages=[
                {
                    "role": "user", 
                    "content": prompt
                }
            ],
        )
        answer = (response.choices[0].message.content or "").strip().strip('"')

        sources = [Source(id=str(result['id']), score=result['score']) for result in results]

        return RagResponse(
            answer=answer,
            sources=sources
        )

    async def add(self, id: uuid.UUID, title: str, description: str) -> Error | None:
        doc = {
            "id": id,
            "title": title,
            "description": description
        }
        async with self._lock:
            if self.hybrid_search.exists(id):
                return Error(f"Document with id {id} already exists.")
            
            await asyncio.to_thread(self.hybrid_search.add_document, doc)

    async def update(self, id: uuid.UUID, title: str, description: str) -> Error | None:
        doc = {
            "id": id,
            "title": title,
            "description": description
        }
        async with self._lock:
            if not self.hybrid_search.exists(id):
                return Error(f"Document with id {id} does not exist.")
            
            await asyncio.to_thread(self.hybrid_search.update_document, doc)

    async def delete(self, id: uuid.UUID) -> Error | None:

        async with self._lock:
            if not self.hybrid_search.exists(id):
                return Error(f"Document with id {id} does not exist.")

            await asyncio.to_thread(self.hybrid_search.delete_document, id)

rag_service = RagService()