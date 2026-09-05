import json
from openai import OpenAI
from .hybrid_search import HybridSearch
from .semantic_search import ChunkedSemanticSearch
from .keyword_search import InvertedIndex
from .reranking import Reranker

from .search_utils import (
    load_movies, 
    load_test_cases,
    DEFAULT_EVALUATION_LIMIT,
)
from .gen_utils import client, GEN_MODEL


def evaluate_search_command(search_type: str ,limit: int = DEFAULT_EVALUATION_LIMIT, rerank: bool = False) -> None:
    movies = load_movies()
    test_cases = load_test_cases()

    match search_type:
        case "hybrid":
            hs = HybridSearch(movies, test=True)
            search = hs.rrf_search
        case "semantic":
            css = ChunkedSemanticSearch(test=True)
            css.load_or_create_chunk_embeddings(movies)
            search = css.search_chunks
        case "keyword":
            idx = InvertedIndex(test=True)
            idx.load_or_build_index(movies)
            search = idx.bm25_search
        case _:
            raise ValueError(f"{search_type} is not supported")

    if rerank:
        re = Reranker()
        top_k = limit
        limit *= 2

    for test_case in test_cases:
        query: str = test_case["query"]
        relevant_docs: set[str] = set(test_case["relevant_docs"])
        search_results = search(query, limit=limit)

        if rerank:
            search_results = re.rerank_cross_encoder(query, search_results, top_k)
            limit = top_k

        retrieved_docs = []
        for result in search_results:
            title = result.get("title", "")
            if title:
                retrieved_docs.append(title)

        relevant_count = 0
        for doc in retrieved_docs:
            if doc in relevant_docs:
                relevant_count += 1

        precision = relevant_count / len(retrieved_docs)
        recall = relevant_count / len(relevant_docs)
        if precision + recall == 0:
            f1_socre = 0
        else:
            f1_socre = 2 * (precision * recall) / (precision + recall)

        print(f"- Query: {query}")
        print(f"    - Precision@{limit}: {precision:.4f}")
        print(f"    - Recall@{limit}: {recall:.4f}")
        print(f"    - F1 Score: {f1_socre:.4f}")
        print(f"    - Retreived: {', '.join(retrieved_docs)}")
        print(f"    - Relevant: {', '.join(relevant_docs)}")
        print()

def evaluate_with_llm(query: str, results: list[dict]) -> list[dict]:
    formatted_results = []
    for i, result in enumerate(results, 1):
        formatted_results.append(f"{i}. {result['title']}")
    
    prompt = f"""Rate how relevant each result is to this query on a 0-3 scale:

    Query: "{query}"

    Results:
    {"\n".join(formatted_results)}

    Scale:
    - 3: Highly relevant
    - 2: Relevant
    - 1: Marginally relevant
    - 0: Not relevant

    Do NOT give any numbers other than 0, 1, 2, or 3.

    Return ONLY the scores in the same order you were given the documents. Return a valid JSON list, nothing else. For example:

    [2, 0, 3, 2, 0, 1]"""

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
    ratings: list[int] = json.loads(corrected)

    if len(ratings) != len(results):
        raise ValueError(
            f"LLM response parsing error. Expected {len(results)} scores, got {len(ratings)}. Response: {ratings}"
        )

    evaluations = []
    for i, result in enumerate(results):
        evaluations.append({"title": result["title"], "rating": ratings[i]})
    return evaluations