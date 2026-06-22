#!/usr/bin/env python3
"""Static checks for the production deployment release pipeline."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    full_path = ROOT / path
    if not full_path.exists():
        raise AssertionError(f"missing required file: {path}")
    return full_path.read_text(encoding="utf-8")


def require(text: str, needle: str, label: str) -> None:
    if needle not in text:
        raise AssertionError(f"{label}: expected to find {needle!r}")


def forbid(text: str, needle: str, label: str) -> None:
    if needle in text:
        raise AssertionError(f"{label}: must not contain {needle!r}")


def check_prod_compose() -> None:
    compose = read("docker-compose.prod.yml")
    require(compose, "ghcr.io/billionsheep/agent-imageflow-api:${IMAGE_TAG:-main}", "prod compose api image")
    require(compose, "ghcr.io/billionsheep/agent-imageflow-web:${IMAGE_TAG:-main}", "prod compose web image")
    require(compose, 'command: ["/app/api"]', "prod compose api command")
    require(compose, 'command: ["/app/worker"]', "prod compose worker command")
    require(compose, "${AGENT_IMAGEFLOW_STORAGE_ROOT:-asset-storage}", "prod compose storage root override")
    require(compose, "PUBLIC_BASE_URL: ${PUBLIC_BASE_URL:?set PUBLIC_BASE_URL}", "prod compose public base")
    forbid(compose, "build:", "prod compose")
    postgres_block = re.search(r"(?ms)^  postgres:\n(?P<body>.*?)(?:\n  [a-z][a-z0-9_-]+:|\nvolumes:)", compose)
    redis_block = re.search(r"(?ms)^  redis:\n(?P<body>.*?)(?:\n  [a-z][a-z0-9_-]+:|\nvolumes:)", compose)
    if not postgres_block or not redis_block:
        raise AssertionError("prod compose: postgres and redis services are required")
    forbid(postgres_block.group("body"), "ports:", "prod compose postgres")
    forbid(redis_block.group("body"), "ports:", "prod compose redis")


def check_web_image() -> None:
    dockerfile = read("Dockerfile.web")
    require(dockerfile, "FROM node:22-alpine AS build", "web Dockerfile build image")
    require(dockerfile, "npm --prefix web ci", "web Dockerfile deterministic install")
    require(dockerfile, "npm --prefix web run build", "web Dockerfile build")
    require(dockerfile, "FROM nginx:1.27-alpine", "web Dockerfile runtime image")
    require(dockerfile, "COPY docker/nginx-web.conf", "web Dockerfile nginx config")
    require(dockerfile, "COPY --from=build /src/web/dist /usr/share/nginx/html", "web Dockerfile dist copy")
    nginx = read("docker/nginx-web.conf")
    require(nginx, "try_files $uri $uri/ /index.html;", "nginx SPA fallback")
    require(nginx, "listen 8080;", "nginx unprivileged port")


def check_github_actions() -> None:
    workflow = read(".github/workflows/docker-publish.yml")
    require(workflow, "ghcr.io/billionsheep/agent-imageflow-api", "workflow api image")
    require(workflow, "ghcr.io/billionsheep/agent-imageflow-web", "workflow web image")
    require(workflow, "npm --prefix web test -- --run", "workflow web tests")
    require(workflow, "npm --prefix web run build", "workflow web build")
    require(workflow, "golang:1.25.3-alpine", "workflow containerized go")
    require(workflow, "/usr/local/go/bin/go test ./...", "workflow pinned go binary")
    require(workflow, "docker/build-push-action", "workflow docker build-push")
    require(workflow, "push: ${{ github.event_name != 'pull_request' }}", "workflow no push on PR")
    require(workflow, "type=sha,prefix=sha-,format=short", "workflow sha tag")
    require(workflow, "type=ref,event=tag", "workflow release tag")
    forbid(workflow, "OPENAI_COMPATIBLE_API_KEY", "workflow")
    forbid(workflow, "FAL_API_KEY", "workflow")


def check_env_example() -> None:
    env = read(".env.example.prod")
    for key in [
        "IMAGE_TAG=",
        "PUBLIC_BASE_URL=",
        "POSTGRES_PASSWORD=",
        "ADMIN_USERNAME=",
        "ADMIN_PASSWORD=",
        "ADMIN_SESSION_SECRET=",
        "BASIC_AUTH_USERNAME=",
        "BASIC_AUTH_PASSWORD=",
        "OPENAI_COMPATIBLE_BASE_URL=",
        "OPENAI_COMPATIBLE_API_KEY=",
        "OPENAI_COMPATIBLE_MODEL=",
        "AGENT_IMAGEFLOW_STORAGE_ROOT=",
    ]:
        require(env, key, "prod env example")
    for unsafe in ["sk-", "test-key", "admin:secret"]:
        forbid(env.lower(), unsafe, "prod env example")


def main() -> int:
    checks = [
        check_prod_compose,
        check_web_image,
        check_github_actions,
        check_env_example,
    ]
    for check in checks:
        check()
    print("deployment release pipeline static checks passed")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except AssertionError as exc:
        print(f"deployment release pipeline check failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
