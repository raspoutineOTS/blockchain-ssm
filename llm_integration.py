def translate_norm(norm_text, context):
    prompt = f"Translate the following norm into context-appropriate language for {context}:\\n\\n{norm_text}"
    translated_norm = "Translated Norm"  # Replace with actual LLM processing
    return translated_norm

def adapt_norm_to_context(norm, context_factors):
    adaptation_prompt = f"Adapt the following norm to the given context factors:\\nNorm: {norm}\\nContext Factors: {context_factors}"
    adapted_norm = "Adapted Norm"  # Replace with actual LLM processing
    return adapted_norm
