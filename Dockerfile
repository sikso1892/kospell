# syntax=docker/dockerfile:1
ARG GO_VERSION=1.24.1

# ── build ─────────────────────────────────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates git && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/kospell-server ./cmd/kospell-server

# ── runtime (single image, mode controlled by ENV / CLI) ──────────────────────
FROM alpine:3.21

# ca-certificates : HTTPS (nara-speller, OpenAI)
# hunspell        : used only when MODE=hunspell
RUN apk add --no-cache ca-certificates tzdata hunspell && update-ca-certificates
RUN mkdir -p /dict

COPY --from=builder /out/kospell-server /usr/local/bin/kospell-server
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# ── environment defaults ───────────────────────────────────────────────────────
# MODE         : nara | hunspell | openai
# PORT         : HTTP listen port
# DICT_DIR     : hunspell dictionary directory  (MODE=hunspell)
# DICT_LANG    : hunspell dictionary name       (MODE=hunspell, default: ko)
# OPENAI_API_KEY : API key                      (MODE=openai)
# LLM_MODEL    : model name                     (MODE=openai)
# LLM_BASE_URL : custom OpenAI-compatible URL   (MODE=openai)
ENV MODE=nara \
    PORT=8080 \
    DICT_DIR=/dict \
    DICT_LANG=ko \
    OPENAI_API_KEY="" \
    LLM_MODEL="" \
    LLM_BASE_URL=""

EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
