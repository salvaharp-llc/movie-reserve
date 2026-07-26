from pydantic import BaseModel

class RagRequest(BaseModel):
    query: str
    top_k: int = 5