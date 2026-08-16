from .gen_utils import async_client, GEN_MODEL
from .search_utils import DEFAULT_SEARCH_LIMIT
from .hybrid_search import HybridSearch
from .reranking import Reranker
from .query_enhancement import async_enhance_query

def build_prompt(query: str, docs: str, answer_type: str = "search") -> str | None:
    match answer_type:
        case "search":
            return f"""You are a RAG agent for Hoopla, a movie streaming service.
            Your task is to provide a natural-language answer to the user's query based on documents retrieved during search.
            Provide a comprehensive answer that addresses the user's query.

            Query: {query}

            Documents:
            {docs}

            Answer:"""
        case "summary":
            return f"""Provide information useful to the query below by synthesizing data from multiple search results in detail.

            The goal is to provide comprehensive information so that users know what their options are.
            Your response should be information-dense and concise, with several key pieces of information about the genre, plot, etc. of each movie.

            This should be tailored to Hoopla users. Hoopla is a movie streaming service.

            Query: {query}

            Search results:
            {docs}

            Provide a comprehensive 3–4 sentence answer that combines information from multiple sources:"""
        case "citation":
            return f"""Answer the query below and give information based on the provided documents.

            The answer should be tailored to users of Hoopla, a movie streaming service.
            If not enough information is available to provide a good answer, say so, but give the best answer possible while citing the sources available.

            Query: {query}

            Documents:
            {docs}

            Instructions:
            - Provide a comprehensive answer that addresses the query
            - Cite sources in the format [1], [2], etc. when referencing information
            - If sources disagree, mention the different viewpoints
            - If the answer isn't in the provided documents, say "I don't have enough information"
            - Be direct and informative

            Answer:"""
        case "question_answer":
            return f"""Answer the user's question based on the provided movies that are available on Hoopla, a streaming service.

            Question: {query}

            Documents:
            {docs}

            Instructions:
            - Answer questions directly and concisely
            - Be casual and conversational
            - Don't be cringe or hype-y
            - Talk like a normal person would in a chat conversation
            - Use only information from the documents
            - If the answer isn't in the documents, say "I don't have enough information"
            - Cite sources when possible

            Guidance on types of questions:
            - Factual questions: Provide a direct answer
            - Analytical questions: Compare and contrast information from the documents
            - Opinion-based questions: Acknowledge subjectivity and provide a balanced view
            Answer:"""
        case _:
            return None

def format_search_results(results: list[dict]) -> str:
    formatted_results = []
    for result in results:
        # formatted_results.append(f"{result['title']}: {result['description']}") results' descriptions too long of a prompt for the LLM to handle, so just include the title
        formatted_results.append(result['title'])
    return '\n\n'.join(formatted_results)