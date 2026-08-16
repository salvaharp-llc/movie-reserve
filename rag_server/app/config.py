from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    APP_NAME: str = "Movie RAG Service"
    VERSION: str = "1.0.0"
    OPENROUTER_API_KEY: str
    API_KEY: str
    DB_URL: str
    TRANSFORMER_MODEL: str

    class Config:
        env_file = ".env"

settings = Settings()