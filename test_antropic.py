import os
import anthropic
import dotenv
import time
import subprocess
import signal
import sys

dotenv.load_dotenv()
api_key = os.getenv("REQUESTY_API_KEY")

if not api_key:
    print("\033[34Error: REQUESTY_API_KEY not found in environment variables")
    sys.exit(1)


def print_err(message):
    print(f"\033[31m{message}\033[0m")


def print_ok(message):
    print(f"\033[32m{message}\033[0m")


def print_info(message):
    print(f"\033[34m{message}\033[0m")

def test_anthropic_api():
    """Test the Anthropic API through the gateway"""
    client = anthropic.Anthropic(
        api_key=api_key,
        base_url="https://ngoowcoo0kg0gowgs4okccw0.levy42.com/",
    )
    print_info("Sending a test request...")
    message = client.messages.create(
        max_tokens=1024,
        model="openai/gpt-4o-mini",
        messages=[
            {"role": "user", "content": "Hello, Claude!"}
        ]
    )
    print_ok(f"Response: {message.content[0].text}")

    print_info("Sending a streaming test request...")
    with client.messages.stream(
            model="openai/gpt-4o-mini",
            max_tokens=1000,
            messages=[
                {"role": "user", "content": "Hello, Claude! Name 10 largest countires"}
            ]
    ) as stream:
        print_info("Streaming response:")
        for text in stream.text_stream:
            print(f"{text}", end="", flush=True)
        print()
        print_ok(f"Streaming finished")


if __name__ == "__main__":
    test_anthropic_api()
    print_ok("Test completed.")
