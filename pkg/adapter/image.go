package adapter

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func parseImageURL(imageURL string) ([]byte, string, error) {
	if strings.HasPrefix(imageURL, "data:image/") {
		return decodeBase64Image(imageURL)
	}
	return getImageInfoFromURL(imageURL)
}

func decodeBase64Image(base64String string) ([]byte, string, error) {
	// get image format
	format, err := getBase64ImageFormat(base64String)
	if err != nil {
		return nil, "", err
	}

	// Remove the base64 prefix (e.g., "data:image/png;base64,") if present
	base64String = strings.TrimPrefix(base64String, "data:image/")
	index := strings.Index(base64String, ";base64,")
	if index != -1 {
		base64String = base64String[index+len(";base64,"):]
	}

	// Decode base64 string to byte slice
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, "", err
	}

	return data, format, nil
}

func getBase64ImageFormat(dataURI string) (string, error) {
	// Find the index of "image/"
	startIndex := strings.Index(dataURI, "image/")
	if startIndex == -1 {
		return "", fmt.Errorf("image format not found in data URI")
	}

	// Extract the substring between "image/" and ";"
	startIndex += len("image/")
	endIndex := strings.Index(dataURI[startIndex:], ";")
	if endIndex == -1 {
		return "", fmt.Errorf("image format not found in data URI")
	}

	return dataURI[startIndex : startIndex+endIndex], nil
}

func getImageInfoFromURL(url string) ([]byte, string, error) {
	// Make an HTTP GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	// Read the response body
	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}

	// Extract image format from the "Content-Type" header
	contentType := response.Header.Get("Content-Type")
	format, err := getImageFormatFromContentType(contentType)
	if err != nil {
		return nil, "", err
	}

	return imageData, format, nil
}

func getImageFormatFromContentType(contentType string) (string, error) {
	// Extract image format from the "Content-Type" header
	parts := strings.Split(contentType, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Content-Type header")
	}
	return parts[1], nil
}
