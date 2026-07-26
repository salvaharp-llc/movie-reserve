from openai import OpenAI

from .gen_utils import client, GEN_MODEL

enhancement_prompts = {
    # Spell Enhancement
    "spell": """Fix any spelling errors in the user-provided movie search query below.
    Correct only clear, high-confidence typos. Do not rewrite, add, remove, or reorder words.
    Preserve punctuation and capitalization unless a change is required for a typo fix.
    If there are no spelling errors, or if you're unsure, output the original query unchanged.
    Output only the final query text, nothing else.
    """,
    # Rewrite Enhancement
    "rewrite": """Rewrite the user-provided movie search query below to be more specific and searchable.

    Consider:
    - Common movie knowledge (famous actors, popular films)
    - Genre conventions (horror = scary, animation = cartoon)
    - Keep the rewritten query concise (under 10 words)
    - It should be a Google-style search query, specific enough to yield relevant results
    - Don't use boolean logic

    Examples:
    - "that bear movie where leo gets attacked" -> "The Revenant Leonardo DiCaprio bear attack"
    - "movie about bear in london with marmalade" -> "Paddington London marmalade"
    - "scary movie with bear from few years ago" -> "bear horror movie 2015-2020"

    If you cannot improve the query, output the original unchanged.
    Output only the rewritten query text, nothing else.
    """,
    # Expand Enhancement
    "expand": """Expand the user-provided movie search query below with related terms.

    Add synonyms and related concepts that might appear in movie descriptions.
    Keep expansions relevant and focused.
    Keep the original terms in the modified query.
    Output only the final query text, nothing else. 

    Examples:
    - "scary bear movie" -> "scary horror grizzly bear terrifying film movie"
    - "action movie with bear" -> "action movie thriller with bear chase fight adventure"
    - "comedy with bear" -> "funny comedy with bear lighthearted humor "
    """,
}

def enhance_query(query: str, option: str) -> str:
    if option not in enhancement_prompts:
        return query
    
    prompt = f"""{enhancement_prompts[option]}
    User query: "{query}"
    """
    
    response = client.chat.completions.create(
        model=GEN_MODEL,
        messages=[
            {
                "role": "user",
                "content": prompt
            }
        ]
    )
    corrected = (response.choices[0].message.content or "").strip().strip('"')
    return corrected if corrected else query
