# AI Gateway

A simple proxy gateway that translates Anthropic API requests to OpenAI-compatible API format.

## step 1

Added converter.go and request.go modules to handle non-streaming requests

## step 2

Added streaming.go to support streaming requests.
OpenAI and Anthropic both use SSE for streaming, but they use different formats for messages.

Also added token_counter, which only uses openai tokenizers.
This is needed for streaming as OpenAI doesn't provide usage metadata for streaming requests.

## step 3
...

## step 4

I've deployed it to my hobby Hetzner instance where I run docker and traefik

I might misunderstand logging request - I implemented it on application level,
as it makes sense to track response/request and metadata having more control over it.

## step 5
Added db cache for /v1/models pricing data
Use it to calculate total cost for each request.

## Testing:

Run test locally
> python test_anthropic.py

Open https://ngoowcoo0kg0gowgs4okccw0.levy42.com/
