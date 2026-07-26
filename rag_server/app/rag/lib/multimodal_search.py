import os
import numpy as np

from PIL import Image
from sentence_transformers import SentenceTransformer

from .semantic_search import cosine_similarity
from .search_utils import load_movies, DEFAULT_SEARCH_LIMIT, CACHE_DIR


class MultimodalSearch():
    def __init__(self, documents: list[dict] = [], model_name: str = "clip-ViT-B-32"):
        self.documents = documents
        self.texts = []
        for doc in documents:
            self.texts.append(f"{doc['title']}: {doc['description']}")

        self.model = SentenceTransformer(model_name)
        self.text_embeddings_path = os.path.join(CACHE_DIR, "multimodal_text_embeddings.npy")
        self.text_embeddings = self.load_or_create_text_embeddings()  

    def load_or_create_text_embeddings(self):
        if os.path.exists(self.text_embeddings_path):
            text_embeddings = np.load(self.text_embeddings_path)
            if len(text_embeddings) == len(self.documents):
                return text_embeddings
            
        return self.build_text_embeddings()
    
    def build_text_embeddings(self):
        text_embeddings = self.model.encode(self.texts, show_progress_bar=True)

        os.makedirs(CACHE_DIR, exist_ok=True)
        np.save(self.text_embeddings_path, text_embeddings)
        return text_embeddings 

    def embed_image(self, image_path: str):
        if not os.path.exists(image_path):
            raise FileNotFoundError(f"Image file not found: {image_path}")
        img = Image.open(image_path)
        image_embedding = self.model.encode([img])
        return image_embedding[0]
    
    def search_with_image(self, image_path: str) -> list[dict]:
        image_embedding = self.embed_image(image_path)
        similarities = []
        for i, text_embedding in enumerate(self.text_embeddings):
            similarities.append((i, cosine_similarity(image_embedding, text_embedding)))
        similarities.sort(key=lambda x: x[1], reverse=True)

        results = []
        for idx, score in similarities[:DEFAULT_SEARCH_LIMIT]:
            doc = self.documents[idx]
            results.append({**doc, 'similarity_score': score})
        return results

        

    
def verify_image_embedding(image_path: str) -> None:
    ms = MultimodalSearch()
    embedding = ms.embed_image(image_path)
    print(f"Embedding shape: {embedding.shape[0]} dimensions")

def image_search_command(image_path: str) -> list[dict]:
    movies = load_movies()
    ms = MultimodalSearch(movies)
    return ms.search_with_image(image_path)