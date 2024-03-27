# Gemini-OpenAI-Proxy

Gemini-OpenAI-Proxy is a proxy designed to convert the OpenAI API protocol to the Google Gemini Pro protocol. This enables seamless integration of OpenAI-powered functionalities into applications using the Gemini Pro protocol.

---

## Table of Contents

- [Gemini-OpenAI-Proxy](#gemini-openai-proxy)
  - [Table of Contents](#table-of-contents)
  - [Build](#build)
  - [Deploy](#deploy)
  - [Usage](#usage)
  - [Compatibility Testing](#compatibility-testing)
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

3. **Integrate the Proxy into Your Application:**
   Modify your application's API requests to target the Gemini-OpenAI-Proxy, providing the acquired Google AI Studio API key as if it were your OpenAI API key.

   Example API Request (Assuming the proxy is hosted at `http://localhost:8080`):
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

   Alternatively, use Gemini Pro Vision:

   ```bash
   curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
        "model": "gpt-4-vision-preview",
        "messages": [{"role": "user", "content": [
           {"type": "text", "text": "What’s in this image?"},
           {
             "type": "image_url",
             "image_url": {
               "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"
             }
           }
        ]}],
        "temperature": 0.7
    }'
   ```

   If you already have access to the Gemini 1.5 Pro api, you can use:

   ```bash
   curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
        "model": "gpt-4-turbo-preview",
        "messages": [{"role": "user", "content": "Say this is a test!"}],
        "temperature": 0.7
    }'
   ```

   Model Mapping:

   - gpt-3.5-turbo -> gemini-1.0-pro-latest
   - gpt-4 -> gemini-1.0-ultra-latest
   - gpt-4-turbo-preview -> gemini-1.5-pro-latest
   - gpt-4-vision-preview -> gemini-1.0-pro-vision-latest

   These are the corresponding model mappings for your reference. We've aligned the models from our project with the latest offerings from Gemini, ensuring compatibility and seamless integration.

4. **Handle Responses:**
   Process the responses from the Gemini-OpenAI-Proxy in the same way you would handle responses from OpenAI.

Now, your application is equipped to leverage OpenAI functionality through the Gemini-OpenAI-Proxy, bridging the gap between OpenAI and applications using the Google Gemini Pro protocol.

## Compatibility Testing

Gemini-OpenAI-Proxy is designed to seamlessly integrate OpenAI-powered functionalities into applications using the Google Gemini Pro protocol. To ensure comprehensive compatibility, we have conducted testing specifically targeting `chatbox` and `openai translator` functionalities.

---

## License

Gemini-OpenAI-Proxy is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.