from pydantic import BaseModel
from typing import List

class Source(BaseModel):
    id: str
    score: float

class RagResponse(BaseModel):
    answer: str
    sources: List[Source]