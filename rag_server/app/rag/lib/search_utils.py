import json
import os

DEFAULT_SEARCH_LIMIT = 5
DOCUMENT_PREVIEW_LENGTH = 100

DEFAULT_CHUNK_SIZE = 200
DEFAULT_SEMANTIC_CHUNK_SIZE = 4
DEFAULT_CHUNK_OVERLAP = 1

SEARCH_MULTIPLIER = 5

BM25_K1 = 1.5
BM25_B = 0.75

DEFAULT_ALPHA = 0.5
DEFAULT_K = 60

DEFAULT_EVALUATION_LIMIT = 5

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
DATA_PATH = os.path.join(PROJECT_ROOT, "data", "movies.json")
STOPWORDS_PATH = os.path.join(PROJECT_ROOT, "data", "stopwords.txt")
GOLDEN_DATASET_PATH = os.path.join(PROJECT_ROOT, "data", "golden_dataset.json")

CACHE_DIR = os.path.join(PROJECT_ROOT, "cache")


def load_movies() -> list[dict]:
    with open(DATA_PATH, "r") as f:
        data = json.load(f)
    return data["movies"]


def load_stopwords() -> list[str]:
    with open(STOPWORDS_PATH, "r") as f:
        return f.read().splitlines()
    
def load_test_cases() -> list[dict]:
    with open(GOLDEN_DATASET_PATH, "r") as f:
        golden_dataset = json.load(f)
    return golden_dataset["test_cases"]