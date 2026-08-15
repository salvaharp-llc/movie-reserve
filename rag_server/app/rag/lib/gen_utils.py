import os
from dotenv import load_dotenv
from openai import OpenAI, AsyncOpenAI

load_dotenv()
api_key = os.environ.get("OPENROUTER_API_KEY")
if not api_key:
    raise RuntimeError("OPENROUTER_API_KEY environment variable not set")

base_url = "https://openrouter.ai/api/v1"

GEN_MODEL = "openrouter/free"

client = OpenAI(
    base_url=base_url,
    api_key=api_key,
)

async_client = AsyncOpenAI(
    base_url=base_url,
    api_key=api_key,

)
