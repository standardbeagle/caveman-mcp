package main

import (
	"context"
	"net/url"
)

func extractYouTube(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractGitHub(ctx context.Context, u *url.URL) (string, error) {
	return "", nil // TODO
}

func extractArXiv(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractHN(ctx context.Context, id string) (string, error) {
	return "", nil // TODO
}

func extractReddit(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractRSS(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func DescribeImage(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}

func TranscribeAudio(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}
