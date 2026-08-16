from enum import Enum
from typing import Optional
import uuid

from pydantic import BaseModel

class EnhanceType(Enum):
    NONE = "none"
    SPELL = "spell"
    REWRITE = "rewrite"
    EXPAND = "expand"
    
class RagRequest(BaseModel):
    query: str
    top_k: Optional[int] = None
    rerank: bool = False
    enhance: EnhanceType = EnhanceType.NONE

class AddRequest(BaseModel):
    id: uuid.UUID
    title: str
    description: str

class UpdateRequest(BaseModel):
    title: str
    description: str