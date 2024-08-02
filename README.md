# Gemini-OpenAI-Proxy

Gemini-OpenAI-Proxy is a proxy designed to convert the OpenAI API protocol to the Google Gemini protocol. This enables applications built for the OpenAI API to seamlessly communicate with the Gemini protocol, including support for Chat Completion, Embeddings, and Model(s) endpoints.

This is a fork of zhu327/gemini-openai-proxy that eliminates the mapping of openAI models to gemini models and directly exposes the underlying gemini models to the api endpoints directly. I've also added support for Google's embeddings model. This was motivated by my own issues with using Google's [openAI API Compatible Endpoint](https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/call-gemini-using-openai-library).

---

## Table of Contents

- [Gemini-OpenAI-Proxy](#gemini-openai-proxy)
  - [Table of Contents](#table-of-contents)
  - [Build](#build)
  - [Deploy](#deploy)
  - [Usage](#usage)
  - [Compatibility](#compatibility)
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

   You can either do this on the command line:
   ```bash
   docker run --restart=unless-stopped -it -d -p 8080:8080 --name gemini ghcr.io/ekatiyar/gemini-openai-proxy:latest
   ```

   Or with the following docker-compose config:
   ```yaml
   version: '3'
   services:
      gemini:
         container_name: gemini
         ports:
            - "8080:8080"
         image: ghcr.io/ekatiyar/gemini-openai-proxy:latest
         restart: unless-stopped
   ```

Adjust the port mapping (e.g., `-p 5001:8080`) as needed, and ensure that the Docker image version aligns with your requirements. If you only want the added embedding model support and still want open ai model mapping, use `ghcr.io/ekatiyar/gemini-openai-proxy:embedding` instead

---

## Usage

Gemini-OpenAI-Proxy offers a straightforward way to integrate OpenAI functionalities into any application that supports custom OpenAI API endpoints. Follow these steps to leverage the capabilities of this proxy:

1. **Set Up OpenAI Endpoint:**
   Ensure your application is configured to use a custom OpenAI API endpoint. Gemini-OpenAI-Proxy seamlessly works with any OpenAI-compatible endpoint.

2. **Get Google AI Studio API Key:**
   Before using the proxy, you'll need to obtain an API key from [ai.google.dev](https://ai.google.dev). Treat this API key as your OpenAI API key when interacting with Gemini-OpenAI-Proxy.

3. **Integrate the Proxy into Your Application:**
   Modify your application's API requests to target the Gemini-OpenAI-Proxy, providing the acquired Google AI Studio API key as if it were your OpenAI API key.

   Example Chat Completion API Request (Assuming the proxy is hosted at `http://localhost:8080`):
   ```bash
   curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
        "model": "gemini-1.0-pro-latest",
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
        "model": "gemini-1.5-vision-latest",
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
   If you wish to map `gemini-1.5-vision-latest` to `gemini-1.5-pro-latest`, you can configure the environment variable `GEMINI_VISION_PREVIEW = gemini-1.5-pro-latest`. This is because `gemini-1.5-pro-latest` now also supports multi-modal data. Otherwise, the default is to use the `gemini-1.5-flash-latest` model

   If you already have access to the Gemini 1.5 Pro api, you can use:

   ```bash
   curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
        "model": "gemini-1.5-pro-latest",
        "messages": [{"role": "user", "content": "Say this is a test!"}],
        "temperature": 0.7
    }'
   ```

   Example Embeddings API Request:

   ```bash
   curl http://localhost:8080/v1/embeddings \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
       "model": "text-embedding-004",
       "input": "This is a test sentence."
    }'
   ```

   You can also pass in multiple input strings as a list:

   ```bash
   curl http://localhost:8080/v1/embeddings \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $YOUR_GOOGLE_AI_STUDIO_API_KEY" \
    -d '{
       "model": "text-embedding-004",
       "input": ["This is a test sentence.", "This is another test sentence"]
    }'
   ```


4. **Handle Responses:**
   Process the responses from the Gemini-OpenAI-Proxy in the same way you would handle responses from OpenAI.

Now, your application is equipped to leverage OpenAI functionality through the Gemini-OpenAI-Proxy, bridging the gap between OpenAI and applications using the Google Gemini Pro protocol.

## Compatibility

- <https://github.com/zhu327/gemini-openai-proxy/issues/4>

---

## License

Gemini-OpenAI-Proxy is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.