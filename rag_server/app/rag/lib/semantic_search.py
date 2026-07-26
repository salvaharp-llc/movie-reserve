import numpy as np
import os
import re
import json
from sentence_transformers import SentenceTransformer

from .search_utils import (
    CACHE_DIR,
    DEFAULT_CHUNK_SIZE,
    DEFAULT_CHUNK_OVERLAP,
    DEFAULT_SEARCH_LIMIT,
    DEFAULT_SEMANTIC_CHUNK_SIZE,
    load_movies,
)

class SemanticSearch:
    def __init__(self) -> None:
        self.model = SentenceTransformer('all-MiniLM-L6-v2')
        self.documents: list[dict] = None
        self.document_map: dict[int, dict] = {}
        self.embeddings = None
        self.embeddings_path = os.path.join(CACHE_DIR, "movie_embeddings.npy")

    def generate_embedding(self, text: str):
        if not text or not text.strip():
            raise ValueError("text can not be an empty string")
        return self.model.encode([text])[0]
    
    def build_embeddings(self):
        documents_strings: list[str] = []
        for doc in self.documents:
            documents_strings.append(f"{doc['title']}: {doc['description']}")
        self.embeddings = self.model.encode(documents_strings, show_progress_bar=True)

        os.makedirs(CACHE_DIR, exist_ok=True)
        np.save(self.embeddings_path, self.embeddings)
        return self.embeddings 
    
    def load_or_create_embeddings(self, documents: list[dict]):
        self.documents = documents
        self.document_map = {}
        for doc in self.documents:
            self.document_map[doc["id"]] = doc

        if os.path.exists(self.embeddings_path):
            self.embeddings = np.load(self.embeddings_path)
            if len(self.embeddings) == len(documents):
                return self.embeddings
            
        return self.build_embeddings()
    
    def search(self, query: str, limit: int = DEFAULT_SEARCH_LIMIT):
        if self.embeddings is None or self.embeddings.size == 0:
            raise ValueError("No embeddings loaded. Call `load_or_create_embeddings` first.")
        
        if self.documents is None or len(self.documents) == 0:
            raise ValueError("No documents loaded. Call `load_or_create_embeddings` first.")
        
        query_embedding = self.generate_embedding(query)
        scores: list[tuple[float, dict]] = []
        for i in range(len(self.documents)):
            similarity_score = cosine_similarity(query_embedding, self.embeddings[i])
            scores.append((similarity_score, self.documents[i]))

        scores.sort(key=lambda x: x[0], reverse=True)
        if len(scores) > limit:
            scores = scores[:limit]

        results: list[dict] = []
        for score, doc in scores:
            result = {
                "score": score,
                "title": doc["title"],
                "description": doc["description"],
            }
            results.append(result)
        return results

class ChunkedSemanticSearch(SemanticSearch):
    def __init__(self) -> None:
        super().__init__()
        self.chunk_embeddings: np.ndarray = None
        self.chunk_metadata: list[dict] = None
        self.chunk_embeddings_path = os.path.join(CACHE_DIR, "chunk_embeddings.npy")
        self.chunk_metadata_path = os.path.join(CACHE_DIR, "chunk_metadata.json")

    def build_chunk_embeddings(self) -> np.ndarray:
        chunks: list[str] = []
        chunks_metadata: list[dict] = []
        for doc in self.documents:
            description = doc["description"].strip()
            if not description:
                continue

            doc_chunks = semantic_chunking(description, 4, 1)
            chunks.extend(doc_chunks)
            total_chunks = len(doc_chunks)
            for i in range(total_chunks):
                chunks_metadata.append(
                    {
                        "movie_idx": doc["id"],
                        "chunk_idx": i,
                        "total_chunks": total_chunks,
                    }
                )
        self.chunk_embeddings = self.model.encode(chunks, show_progress_bar=True)
        self.chunk_metadata = chunks_metadata

        os.makedirs(CACHE_DIR, exist_ok=True)
        np.save(self.chunk_embeddings_path, self.chunk_embeddings)
        with open(self.chunk_metadata_path, "w") as f:
            json.dump({"chunks": self.chunk_metadata, "total_chunks": len(chunks)}, f, indent=2)

        return self.chunk_embeddings
    
    def load_or_create_chunk_embeddings(self, documents: list[dict]) -> np.ndarray:
        self.documents = documents
        self.document_map = {}
        for doc in self.documents:
            self.document_map[doc["id"]] = doc

        if os.path.exists(self.chunk_embeddings_path):
            self.chunk_embeddings = np.load(self.chunk_embeddings_path)
            with open(self.chunk_metadata_path, "r") as f:
                self.chunk_metadata = json.load(f)["chunks"]
            return self.chunk_embeddings
            
        return self.build_chunk_embeddings()
    
    def search_chunks(self, query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
        if self.chunk_embeddings is None or self.chunk_metadata is None:
            raise ValueError(
                "No chunk embeddings loaded. Call load_or_create_chunk_embeddings first."
            )
        query_embedding = self.generate_embedding(query)
        chunk_scores: list[dict] = []
        for i, chunk_embedding in enumerate(self.chunk_embeddings):
            score = cosine_similarity(query_embedding, chunk_embedding)
            chunk_scores.append(
                {
                    "chunk_idx": i,
                    "movie_idx": self.chunk_metadata[i]["movie_idx"],
                    "score": score,
                }
            )
        movie_scores: dict[int, float] = {}
        for chunk_score in chunk_scores:
            movie_idx = chunk_score["movie_idx"]
            score = chunk_score["score"]
            if movie_idx not in movie_scores or score > movie_scores[movie_idx]:
                movie_scores[movie_idx] = score
        sorted_movies = sorted(movie_scores.items(), key=lambda x: x[1], reverse=True)[:limit]
        results: list[dict] = []
        for doc_id, score in sorted_movies:
            doc = self.document_map[doc_id]
            results.append(
                {
                    "id": doc_id,
                    "title": doc["title"],
                    "description": doc["description"],
                    "score": score,
                }
            )
        return results




def verify_model() -> None:
        ss = SemanticSearch()
        print(f"Model loaded: {ss.model}")
        print(f"Max sequence length: {ss.model.max_seq_length}")

def embed_text(text: str) -> None:
    ss = SemanticSearch()
    embedding = ss.generate_embedding(text)
    print(f"Text: {text}")
    print(f"First 3 dimensions: {embedding[:3]}")
    print(f"Dimensions: {embedding.shape[0]}")

def verify_embeddings() -> None:
    ss = SemanticSearch()
    movies = load_movies()
    embeddings = ss.load_or_create_embeddings(movies)
    print(f"Number of docs:   {len(movies)}")
    print(f"Embeddings shape: {embeddings.shape[0]} vectors in {embeddings.shape[1]} dimensions")

def embed_query_text(query: str) -> None:
    ss = SemanticSearch()
    embedding = ss.generate_embedding(query)
    print(f"Query: {query}")
    print(f"First 3 dimensions: {embedding[:3]}")
    print(f"Shape: {embedding.shape}")

def cosine_similarity(vec1, vec2):
    dot_product = np.dot(vec1, vec2)
    norm1 = np.linalg.norm(vec1)
    norm2 = np.linalg.norm(vec2)

    if norm1 == 0 or norm2 == 0:
        return 0.0

    return dot_product / (norm1 * norm2)

def search_command(query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    ss = SemanticSearch()
    movies = load_movies()
    ss.load_or_create_embeddings(movies)
    return ss.search(query, limit)

def chunk_command(text: str, chunk_size: int = DEFAULT_CHUNK_SIZE, overlap: int = DEFAULT_CHUNK_OVERLAP) -> None:
    if overlap < 0:
        raise ValueError("Overlap value cannot be negative")

    words = text.split()
    chunks = []

    n_words = len(words)
    i = 0
    while i < n_words:
        chunk_words = words[i : i + chunk_size]
        if chunks and len(chunk_words) <= overlap:
            break

        chunks.append(" ".join(chunk_words))
        i += chunk_size - overlap

    print(f"Chunking {len(text)} characters")
    for i, chunk in enumerate(chunks, 1):
        print(f"{i}. {chunk}")

def semantic_chunking(text: str, max_chunk_size: int = DEFAULT_SEMANTIC_CHUNK_SIZE, overlap: int = DEFAULT_CHUNK_OVERLAP) -> list[str]:
    if overlap < 0:
        raise ValueError("Overlap value cannot be negative")
    
    sentences = re.split(r"(?<=[.!?])\s+", text)
    n_sentences = len(sentences)
    if n_sentences == 1 and not sentences[0].endswith((".", "!", "?")):
        sentences = [text]

    chunks = []
    i = 0
    while i < n_sentences:
        chunk_sentences = sentences[i : i + max_chunk_size]
        if chunks and len(chunk_sentences) <= overlap:
            break
        
        cleaned_sentences = []
        for chunk_sentence in chunk_sentences:
            cleaned_sentences.append(chunk_sentence.strip())
        if not cleaned_sentences:
            continue
        chunk = " ".join(cleaned_sentences)
        chunks.append(chunk)
        i += max_chunk_size - overlap
    return chunks

def semantic_chunk_command(text: str, max_chunk_size: int = DEFAULT_SEMANTIC_CHUNK_SIZE, overlap: int = DEFAULT_CHUNK_OVERLAP) -> None:
    chunks = semantic_chunking(text, max_chunk_size, overlap)

    print(f"Semantically chunking {len(text)} characters")
    for i, chunk in enumerate(chunks, 1):
        print(f"{i}. {chunk}")

def embed_chunks_command() -> None:
    movies = load_movies()
    css = ChunkedSemanticSearch()
    embeddings = css.load_or_create_chunk_embeddings(movies)
    print(f"Generated {len(embeddings)} chunked embeddings")

def search_chunked_command(query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    movies = load_movies()
    css = ChunkedSemanticSearch()
    css.load_or_create_chunk_embeddings(movies)
    return css.search_chunks(query, limit)
        