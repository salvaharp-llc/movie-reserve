import os
import pickle
import string
import math
from collections import Counter, defaultdict

from nltk.stem import PorterStemmer

from .search_utils import (
    CACHE_DIR,
    DEFAULT_SEARCH_LIMIT,
    BM25_K1,
    BM25_B,
    load_movies,
    load_stopwords,
)

class InvertedIndex:
    def __init__(self) -> None:
        self.index: defaultdict[str, set[int]] = defaultdict(set)
        self.docmap: dict[int, dict] = {}
        self.doc_lengths: dict[int, int] = {}
        self.term_frequencies: defaultdict[int, Counter] = defaultdict(Counter)
        self.index_path = os.path.join(CACHE_DIR, "index.pkl")
        self.docmap_path = os.path.join(CACHE_DIR, "docmap.pkl")
        self.tf_path = os.path.join(CACHE_DIR, "term_frequencies.pkl")
        self.doc_lengths_path = os.path.join(CACHE_DIR, "doc_lengths.pkl")

    def build(self) -> None:
        movies = load_movies()
        for movie in movies:
            doc_id = movie["id"]
            text = f"{movie["title"]} {movie["description"]}"
            self.__add_document(doc_id, text)
            self.docmap[doc_id] = movie

    def save(self) -> None:
        os.makedirs(CACHE_DIR, exist_ok=True)
        with open(self.index_path, "wb") as f:
            pickle.dump(self.index, f)
        with open(self.docmap_path, "wb") as f:
            pickle.dump(self.docmap, f)
        with open(self.tf_path, "wb") as f:
            pickle.dump(self.term_frequencies, f)
        with open(self.doc_lengths_path, "wb") as f:
            pickle.dump(self.doc_lengths, f)

    def load(self) -> None:
        with open(self.index_path, "rb") as f:
            self.index = pickle.load(f)
        with open(self.docmap_path, "rb") as f:
            self.docmap = pickle.load(f)
        with open(self.tf_path, "rb") as f:
            self.term_frequencies = pickle.load(f)
        with open(self.doc_lengths_path, "rb") as f:
            self.doc_lengths = pickle.load(f)
    
    def get_documents(self, term: str) -> list[int]:
        doc_ids = self.index.get(term, set())
        return sorted(list(doc_ids))
    
    def __add_document(self, doc_id: int, text: str) -> None:
        tokens = tokenize_text(text)
        for token in set(tokens):
            self.index[token].add(doc_id)
        self.term_frequencies[doc_id].update(tokens)
        self.doc_lengths[doc_id] = len(tokens)
    
    def __get_avg_doc_length(self) -> float:
        num_docs = len(self.doc_lengths)
        if num_docs == 0:
            return 0.0
        return sum(self.doc_lengths.values()) / num_docs
    
    def get_tf(self, doc_id: int, term: str) -> int:
        tokens = tokenize_text(term)
        if len(tokens) != 1:
            raise ValueError("term must be a single token")
        token = tokens[0]
        return self.term_frequencies[doc_id][token]
    
    def get_bm25_tf(self, doc_id: int, term: str, k1: float = BM25_K1, b: float = BM25_B) -> float:
        tf = self.get_tf(doc_id, term)
        avg_doc_length = self.__get_avg_doc_length()
        doc_length = self.doc_lengths.get(doc_id, 0)
        if avg_doc_length > 0:
            if doc_length == 0: # avoid division by 0 if b == 1 and doc_length == 0
                return 0
            length_norm = 1 - b + b * (doc_length / avg_doc_length)
        else:
            length_norm = 1
        return (tf * (k1 + 1) / (tf + k1 * length_norm))
    
    def get_idf(self, term: str) -> float:
        tokens = tokenize_text(term)
        if len(tokens) != 1:
            raise ValueError("term must be a single token")
        token = tokens[0]
        doc_count = len(self.docmap)
        term_doc_count = len(self.index[token])
        return math.log((doc_count + 1) / (term_doc_count + 1))
    
    def get_bm25_idf(self, term: str) -> float:
        tokens = tokenize_text(term)
        if len(tokens) != 1:
            raise ValueError("term must be a single token")
        token = tokens[0]
        doc_count = len(self.docmap)
        term_doc_count = len(self.index[token])
        return math.log((doc_count - term_doc_count + 0.5) / (term_doc_count + 0.5) + 1)

    def get_tf_idf(self, doc_id: int, term: str) -> float:
        tf = self.get_tf(doc_id, term)
        idf = self.get_idf(term)
        return tf * idf
    
    def bm25(self, doc_id: int, term: str) -> float:
        bm25_tf = self.get_bm25_tf(doc_id, term)
        bm25_idf = self.get_bm25_idf(term)
        return bm25_tf * bm25_idf
    
    def bm25_search(self, query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
        query_tokens = tokenize_text(query)

        scores: dict[int, float] = {}
        for doc_id in self.docmap:
            score = 0.0
            for token in query_tokens:
                score += self.bm25(doc_id, token)
            scores[doc_id] = score

        results_ids = sorted(scores.items(), key = lambda x: x[1], reverse = True)
        if len(results_ids) > limit:
            results_ids = results_ids[:limit]

        results: list[dict] = []
        for doc_id, score in results_ids:
            doc = self.docmap[doc_id]
            result = {
                "id": doc["id"],
                "title": doc["title"],
                "description": doc["description"],
                "score": score,
            }
            results.append(result)
        return results
        
    
def build_command() -> None:
    idx = InvertedIndex()
    idx.build()
    idx.save()

def search_command(query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    idx = InvertedIndex()
    idx.load()
    query_tokens = tokenize_text(query)
    seen, results = set(), []
    for token in query_tokens:
        doc_ids = idx.get_documents(token)
        for doc_id in doc_ids:
            if doc_id in seen:
                continue
            seen.add(doc_id)
            doc = idx.docmap[doc_id]
            results.append(doc)
            if len(results) >= limit:
                return results
            
    return results

def has_matching_token(query_tokens: list[str], title_tokens: list[str]) -> bool:
    for query_token in query_tokens:
        for title_token in title_tokens:
            if query_token in title_token:
                return True
    return False

def tokenize_text(text: str) -> list[str]:
    text = preprocess_text(text)
    tokens = text.split()
    stopwords = load_stopwords()
    filtered_tokens = filter(lambda token: token not in stopwords, tokens)
    stemmer = PorterStemmer()
    stemmed_tokens = map(lambda token: stemmer.stem(token), filtered_tokens)
    return list(stemmed_tokens)

def preprocess_text(text: str) -> str:
    text = text.lower()
    text = text.translate(str.maketrans("", "", string.punctuation))
    return text

def tf_command(doc_id: int, term: str) -> int:
    idx = InvertedIndex()
    idx.load()
    return idx.get_tf(doc_id, term)

def bm25_tf_command(doc_id: int, term: str, k: float = BM25_K1, b: float = BM25_B) -> float:
    idx = InvertedIndex()
    idx.load()
    return idx.get_bm25_tf(doc_id, term, k, b)

def idf_command(term: str) -> float:
    idx = InvertedIndex()
    idx.load()
    return idx.get_idf(term)

def bm25_idf_command(term: str) -> float:
    idx = InvertedIndex()
    idx.load()
    return idx.get_bm25_idf(term)

def tfidf_command(doc_id: int, term: str) -> float:
    idx = InvertedIndex()
    idx.load()
    return idx.get_tf_idf(doc_id, term)

def bm25search_command(query: str, limit: int = DEFAULT_SEARCH_LIMIT) -> list[dict]:
    idx = InvertedIndex()
    idx.load()
    return idx.bm25_search(query, limit)