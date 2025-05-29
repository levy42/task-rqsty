import requests
import json
import dotenv
import os

dotenv.load_dotenv()

def test_sse_stream():
    url = "http://localhost:8080/v1/messages"
    headers = {
        "Content-Type": "application/json",
        "x-api-key": os.environ['REQUESTY_API_KEY']  # Replace with your actual API key
    }
    
    data = {
        "model": "gpt-4o-mini",
        "max_tokens": 100,
        "messages": [
            {"role": "user", "content": "Hello! Give me a short response."}
        ],
        "stream": True
    }

    print("Making streaming request...")
    response = requests.post(url, headers=headers, json=data, stream=True)
    
    if response.status_code != 200:
        print(f"Error: Status code {response.status_code}")
        print(response.text)
        return

    print("Response headers:", dict(response.headers))
    
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            print(f"Raw line: {line}")
            if line.startswith("data: "):
                data = line[6:]  # Remove "data: " prefix
                try:
                    if data == "[DONE]":
                        print("Stream completed")
                        break
                    json_data = json.loads(data)
                    print(f"Parsed JSON: {json.dumps(json_data, indent=2)}")
                except json.JSONDecodeError as e:
                    print(f"JSON parse error: {e}")

if __name__ == "__main__":
    test_sse_stream()