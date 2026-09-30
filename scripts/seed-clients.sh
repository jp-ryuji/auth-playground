#!/usr/bin/env bash
# Register the three dev Hydra OAuth clients (OV-04).
# Idempotent: creates a missing client and updates one that already exists.
#
# Shared secrets below are local placeholders. M2M-02 allows client_secret_basic
# in development. Production M2M and token exchange use private_key_jwt; this
# script does not generate those keys.
set -euo pipefail

export HYDRA_ADMIN_URL="${HYDRA_ADMIN_URL:-http://127.0.0.1:4445}"
export CLIENT_ID="${CLIENT_ID:-auth-playground-rp}"
export CLIENT_SECRET="${CLIENT_SECRET:-dev-only-interactive-rp-secret}"
export REDIRECT_URI="${REDIRECT_URI:-http://127.0.0.1:8080/auth/callback}"
export OAUTH_M2M_CLIENT_ID="${OAUTH_M2M_CLIENT_ID:-auth-playground-m2m}"
export OAUTH_M2M_SECRET="${OAUTH_M2M_SECRET:-dev-only-m2m-secret}"
export OAUTH_EXCHANGE_CLIENT_ID="${OAUTH_EXCHANGE_CLIENT_ID:-auth-playground-exchange}"
export OAUTH_EXCHANGE_SECRET="${OAUTH_EXCHANGE_SECRET:-dev-only-exchange-secret}"

python3 - <<'PY'
import json, os, sys, time, urllib.error, urllib.request

admin = os.environ["HYDRA_ADMIN_URL"].rstrip("/")

clients = [
    {
        "client_id": os.environ["CLIENT_ID"],
        "client_name": "auth-playground interactive RP",
        "client_secret": os.environ["CLIENT_SECRET"],
        "grant_types": ["authorization_code", "refresh_token"],
        "response_types": ["code"],
        "redirect_uris": [os.environ["REDIRECT_URI"]],
        "scope": "openid offline",
        "token_endpoint_auth_method": "client_secret_basic",
        "skip_consent": False,
    },
    {
        "client_id": os.environ["OAUTH_M2M_CLIENT_ID"],
        "client_name": "auth-playground M2M",
        "client_secret": os.environ["OAUTH_M2M_SECRET"],
        "grant_types": ["client_credentials"],
        "response_types": ["token"],
        "scope": "m2m",
        "token_endpoint_auth_method": "client_secret_basic",
    },
    {
        "client_id": os.environ["OAUTH_EXCHANGE_CLIENT_ID"],
        "client_name": "auth-playground token exchange",
        "client_secret": os.environ["OAUTH_EXCHANGE_SECRET"],
        "grant_types": ["urn:ietf:params:oauth:grant-type:token-exchange"],
        "response_types": ["token"],
        "scope": "exchange",
        "token_endpoint_auth_method": "client_secret_basic",
    },
]

def call(method, path, payload=None):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(admin + path, data=data, method=method)
    if payload is not None:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as err:
        return err.code, err.read().decode()

deadline = time.time() + 60
ready = False
while time.time() < deadline:
    try:
        status, _ = call("GET", "/health/ready")
        if status == 200:
            ready = True
            break
    except urllib.error.URLError:
        pass
    time.sleep(1)

if not ready:
    sys.exit(f"hydra admin not ready at {admin}")

for client in clients:
    cid = client["client_id"]
    status, body = call("GET", f"/admin/clients/{cid}")
    if status == 200:
        method, path = "PUT", f"/admin/clients/{cid}"
    elif status == 404:
        method, path = "POST", "/admin/clients"
    else:
        sys.exit(f"lookup {cid} failed: HTTP {status} {body}")

    status, body = call(method, path, client)
    if status not in (200, 201):
        sys.exit(f"{method} {cid} failed: HTTP {status} {body}")
    grants = ",".join(client["grant_types"])
    print(f"registered {cid} ({grants})")
PY
