# Gemini-OpenAI-Proxy

Gemini-OpenAI-Proxy is a proxy designed to convert the OpenAI API protocol to the Google Gemini Pro protocol. This enables seamless integration of OpenAI-powered functionalities into applications using the Gemini Pro protocol.

---

## Table of Contents

- [Build](#build)
- [Deploy](#deploy)
- [Usage](#usage)
- [License](#license)

---

## Build

To build the Gemini-OpenAI-Proxy, follow these steps:

```bash
go build -o gemini main.go
```

---

## Deploy

We recommend deploying Gemini-OpenAI-Proxy using Docker for a straightforward setup. Follow these steps to deploy with Docker:

```bash
docker run --restart=always -it -d -p 8080:8080 --name gemini zhu327/gemini-openai-proxy:latest
```

Adjust the port mapping (e.g., `-p 8080:8080`) as needed, and ensure that the Docker image version (`zhu327/gemini-openai-proxy:latest`) aligns with your requirements.

---

## Usage

Gemini-OpenAI-Proxy offers a straightforward way to integrate OpenAI functionalities into any application that supports custom OpenAI API endpoints. Follow these steps to leverage the capabilities of this proxy:

1. **Set Up OpenAI Endpoint:**
   Ensure your application is configured to use a custom OpenAI API endpoint. Gemini-OpenAI-Proxy seamlessly works with any OpenAI-compatible endpoint.

2. **Get Google AI Studio API Key:**
   Before using the proxy, you'll need to obtain an API key from [ai.google.dev](https://ai.google.dev). Treat this API key as your OpenAI API key when interacting with Gemini-OpenAI-Proxy.

3. **Integrate Proxy into Your Application:**
   Update your application's API requests to point to the Gemini-OpenAI-Proxy, providing the obtained Google AI Studio API key as if it were your OpenAI API key.

   Example API Request (Assuming the proxy is running on `http://localhost:8080`):
   ```bash
   curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Say this is a test!"}],
        "temperature": 0.7
    }'
   ```

4. **Handle Responses:**
   Process the responses from the Gemini-OpenAI-Proxy in the same way you would handle responses from OpenAI.

Now, your application is equipped to leverage OpenAI functionality through the Gemini-OpenAI-Proxy, bridging the gap between OpenAI and applications using the Google Gemini Pro protocol.

## Compatibility Testing

Gemini-OpenAI-Proxy is designed to seamlessly integrate OpenAI-powered functionalities into applications using the Google Gemini Pro protocol. To ensure comprehensive compatibility, we have conducted testing specifically targeting `chatbox` and `openai translator` functionalities.

---

## License

Gemini-OpenAI-Proxy is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.