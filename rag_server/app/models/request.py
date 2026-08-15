from typing import Optional
import uuid

from pydantic import BaseModel

class RagRequest(BaseModel):
    query: str
    top_k: Optional[int]

class AddRequest(BaseModel):
    id: uuid.UUID
    title: str
    description: str

class UpdateRequest(BaseModel):
    title: str
    description: str