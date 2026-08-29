import os
from typing import Optional
import uuid 

from PIL import ImageFile
from .search_utils import (
    load_movies,
    DEFAULT_ALPHA,
    DEFAULT_SEARCH_LIMIT,
    DEFAULT_K,
    SEARCH_MULTIPLIER,
    SENTENCE_TRANSFORMER
)

from .query_enhancement import enhance_query
from .reranking import rerank_results
from .keyword_search import InvertedIndex
from .semantic_search import ChunkedSemanticSearch

class HybridSearch:
    def __init__(self, documents: list[dict], semantic_encoder: str = SENTENCE_TRANSFORMER, test: bool = False):
        self.semantic_search = ChunkedSemanticSearch(semantic_encoder, test)
        self.semantic_search.load_or_create_chunk_embeddings(documents)

        self.idx = InvertedIndex(test)
        self.idx.load_or_build_index(documents)

    def exists(self, id: uuid.UUID) -> bool:
        return id in self.idx.docmap

    def add_document(self, doc: dict) -> None:
        self.semantic_search.add_document(doc)
        self.idx.add_document(doc)

    def update_document(self, doc: dict) -> None:
        self.semantic_search.update_document(doc)
        self.idx.update_document(doc)

    def delete_document(self, doc_id: uuid.UUID) -> None:
        self.semantic_search.delete_document(doc_id)
        self.idx.delete_document(doc_id)

    def set_image(self, doc_id: uuid.UUID, image: ImageFile) -> None:
        self.semantic_search.set_image(doc_id, image)

    def _bm25_search(self, query, limit):
        self.idx.load()
        return self.idx.bm25_search(query, limit)

    def weighted_search(self, query: str, alpha: float = DEFAULT_ALPHA, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
        keyword_results = self._bm25_search(query, limit * 500)
        semantic_results = self.semantic_search.search_chunks(query, limit * 500)

        norm_keyword_results = normalize_search_results(keyword_results)
        norm_semantic_results = normalize_search_results(semantic_results)

        results: dict[uuid.UUID, dict] = {}
        for result in norm_keyword_results:
            doc_id = result["id"]
            results[doc_id] = {
                "id": doc_id,
                "title": result["title"],
                "description": result["description"],
                "bm25_score": result["normalized_score"],
                "semantic_score": 0.0,
            }

        for result in norm_semantic_results:
            doc_id = result["id"]
            if doc_id in results:
                results[doc_id]["semantic_score"] = result["normalized_score"]
            else:
                results[doc_id] = {
                    "id": doc_id,
                    "title": result["title"],
                    "description": result["description"],
                    "bm25_score": 0.0,
                    "semantic_score": result["normalized_score"],
                }
        
        for doc_id, doc in results.items():
            results[doc_id]["score"] = hybrid_score(doc["bm25_score"], doc["semantic_score"], alpha)

        return sorted(results.values(), key=lambda x: x["score"], reverse=True)[:limit]

    def rrf_search(self, query: str, k: int = DEFAULT_K, limit: int = DEFAULT_SEARCH_LIMIT):
        keyword_results = self._bm25_search(query, limit * 500)
        semantic_results = self.semantic_search.search_chunks(query, limit * 500)

        results: dict[uuid.UUID, dict] = {}
        for i, result in enumerate(keyword_results, 1):
            doc_id = result["id"]
            results[doc_id] = {
                "id": doc_id,
                "title": result["title"],
                "description": result["description"],
                "bm25_rank": i,
            }

        for i, result in enumerate(semantic_results, 1):
            doc_id = result["id"]
            if doc_id in results:
                results[doc_id]["semantic_rank"] = i
            else:
                results[doc_id] = {
                    "id": doc_id,
                    "title": result["title"],
                    "description": result["description"],
                    "semantic_rank": i,
                }
        
        for doc_id, doc in results.items():
            rrf_bm25 = rrf_semantic = 0.0
            if "bm25_rank" in doc:
                rrf_bm25 = rrf_score(doc["bm25_rank"], k)
            if "semantic_rank" in doc:
                rrf_semantic = rrf_score(doc["semantic_rank"], k)
            results[doc_id]["score"] = rrf_bm25 + rrf_semantic

        return sorted(results.values(), key=lambda x: x["score"], reverse=True)[:limit]
    
def normalize_scores(scores: list[float]) -> list[float]:
    n_scores = len(scores)
    if n_scores == 0:
        return []
    
    max_score, min_score = max(scores), min(scores)
    score_range = max_score - min_score

    normalized_scores = [1.0] * n_scores
    if score_range == 0:
        return normalized_scores
    
    for i in range(n_scores):
        normalized_scores[i] = (scores[i] - min_score) / score_range

    return normalized_scores

def hybrid_score(bm25_score: float, semantic_score: float, alpha: float = DEFAULT_ALPHA) -> float:
    return alpha * bm25_score + (1 - alpha) * semantic_score

def rrf_score(rank: int, k: int = DEFAULT_K) -> float:
    return 1 / (k + rank)


def normalize_search_results(results: list[dict]) -> list[dict]:
    scores: list[float] = []
    for result in results:
        scores.append(result["score"])

    normalized: list[float] = normalize_scores(scores)
    for i, result in enumerate(results):
        result["normalized_score"] = normalized[i]

    return results
    
def normalize_command(scores: list[float]) -> None:
    normalized_scores = normalize_scores(scores)
    for score in normalized_scores:
        print(f"* {score:.4f}")

def weighted_search_command(query: str, alpha: float = DEFAULT_ALPHA, limit: int = DEFAULT_SEARCH_LIMIT) -> None:
    movies = load_movies()
    hs = HybridSearch(movies)
    results = hs.weighted_search(query, alpha, limit)
    for i, result in enumerate(results, 1):
        print(f"{i}. {result["title"]}")
        print(f"   Hybrid Score: {result["score"]:.3f}")
        print(f"   BM25: {result["bm25_score"]:.3f}, Semantic: {result["semantic_score"]:.3f}")
        print(f"   {result["description"]}...")

def rrf_search_command(
        query: str, k: int = DEFAULT_K, 
        limit: int = DEFAULT_SEARCH_LIMIT, 
        enhance: Optional[str] = None,
        rerank_method: Optional[str] = None,
    ) -> list[dict]:
    if enhance:
        original_query = query
        query = enhance_query(query, option=enhance)
        print(f"Enhanced query ({enhance}): '{original_query}' -> '{query}'\n")

    movies = load_movies()
    hs = HybridSearch(movies)
    results = hs.rrf_search(query, k, SEARCH_MULTIPLIER * limit if rerank_method else limit)

    if rerank_method:
        print("Results before re-ranking:")
        for i, result in enumerate(results, 1):
            print(f"{i}. {result["title"]}")
            print(f"   RRF Score: {result["score"]:.3f}")
            print(f"   BM25 Rank: {result.get("bm25_rank", -1)}, Semantic: {result.get("semantic_rank", -1)}")
        results = rerank_results(query, results, rerank_method, limit)

    return results