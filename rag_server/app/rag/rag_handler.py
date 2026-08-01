from app.models.response import RagResponse, Source
from app.rag.lib.augmented_generation import (
    HybridSearch, 
    load_movies, 
    format_search_results, 
    build_prompt,
    client,
    GEN_MODEL
)
from app.rag.lib.search_utils import DEFAULT_SEARCH_LIMIT

class RagHandler:

    def __init__(self):
        movies = load_movies()
        self.hybrid_search = HybridSearch(movies)

    async def ask(self, query: str, top_k: int | None = None) -> RagResponse:
        if top_k is None:
            top_k = DEFAULT_SEARCH_LIMIT

        results = self.hybrid_search.rrf_search(query, limit=top_k)
        
        if not results:
            return {
                "search_results": [],
                "error": "No results found",
            }
            
        docs = format_search_results(results)

        prompt = build_prompt(query, docs)
    
        response = client.chat.completions.create(
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

rag_handler = RagHandler()