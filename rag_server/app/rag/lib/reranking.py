from openai import OpenAI
import json

from .gen_utils import client, GEN_MODEL
from .search_utils import DEFAULT_SEARCH_LIMIT

from sentence_transformers import CrossEncoder


def rerank_individual(query: str, results: list[dict], limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    for result in results: # - {result.get("description", "")} descriptions are too long of a prompt for the LLM to handle, so just include the title
        prompt = f"""Rate how well this movie matches the search query.

        Query: "{query}"
        Movie: {result.get("title", "")} 

        Consider:
        - Direct relevance to query
        - User intent (what they're looking for)
        - Content appropriateness

        Rate 0-10 (10 = perfect match).
        Output ONLY the number in your response, no other text or explanation.

        Score:"""

        response = client.chat.completions.create(
            model=GEN_MODEL,
            messages=[
                {
                    "role": "user",
                    "content": prompt
                }
            ]
        )
        corrected = (response.choices[0].message.content or "").strip().strip('"')

        result["rerank_score"] = float(corrected)
    
    return sorted(results, key = lambda x: x["rerank_score"], reverse=True)[:limit]

def rerank_batch(query: str, results: list[dict], limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    if not results:
        return []
    
    res_map = {}
    movies_str = ""
    for res in results:
        doc_id = res["id"]
        res_map[doc_id] = res
        movies_str += f"{doc_id}: {res.get('title', '')}\n" # - {res.get('description', '')}\n" too long for LLM to handle, so just include the title

    prompt = f"""Rank the movies listed below by relevance to the following search query.

    Query: "{query}"

    Movies:
    {movies_str}

    Return ONLY the movie IDs in order of relevance (best match first). Return a valid JSON list, nothing else.

    For example:
    [75, 12, 34, 2, 1]

    Ranking:"""

    response = client.chat.completions.create(
        model=GEN_MODEL,
        messages=[
            {
                "role": "user",
                "content": prompt
            }
        ]
    )
    corrected = (response.choices[0].message.content or "").strip().strip('"')
    ranked_ids: list[int] = json.loads(corrected)

    reranked = []
    for i, doc_id in enumerate(ranked_ids, 1):
        if doc_id in res_map:
            reranked.append({**res_map[doc_id], "rerank_rank": i})

    return reranked[:limit]

def rerank_cross_encoder(query: str, results: list[dict], limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    cross_encoder = CrossEncoder("cross-encoder/ms-marco-TinyBERT-L2-v2")

    pairs = []
    for res in results:
        pairs.append([query, f"{res.get('title', '')} - {res.get('description', '')}"])

    scores = cross_encoder.predict(pairs)

    for i, score in enumerate(scores):
        results[i]["cross_encoder_score"] = score

    return sorted(results, key = lambda x: x["cross_encoder_score"], reverse=True)[:limit]

def rerank_results(query: str, results: list[dict], option: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    match option:
        case "individual":
            return rerank_individual(query, results, limit)
        case "batch":
            return rerank_batch(query, results, limit)
        case "cross_encoder":
            return rerank_cross_encoder(query, results, limit)
        case _:
            return results[:limit]