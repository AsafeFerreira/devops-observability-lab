#!/usr/bin/env python3
"""Minimal local webhook that records Alertmanager fired/resolved evidence."""

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
from threading import Lock
from datetime import datetime, timezone


ALERT_FILE = Path(os.getenv("ALERT_FILE", "/data/alerts.ndjson"))
WRITE_LOCK = Lock()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/health":
            self.send_error(404)
            return
        self._json(200, {"status": "ready"})

    def do_POST(self):
        if self.path != "/alerts":
            self.send_error(404)
            return
        try:
            size = min(int(self.headers.get("Content-Length", "0")), 1_048_576)
            payload = json.loads(self.rfile.read(size))
        except (ValueError, json.JSONDecodeError):
            self._json(400, {"error": "invalid JSON"})
            return

        record = {
            "receivedAt": datetime.now(timezone.utc).isoformat(),
            "status": payload.get("status"),
            "groupLabels": payload.get("groupLabels", {}),
            "alerts": [
                {
                    "status": alert.get("status"),
                    "labels": alert.get("labels", {}),
                    "annotations": alert.get("annotations", {}),
                    "startsAt": alert.get("startsAt"),
                    "endsAt": alert.get("endsAt"),
                }
                for alert in payload.get("alerts", [])
            ],
        }
        ALERT_FILE.parent.mkdir(parents=True, exist_ok=True)
        with WRITE_LOCK, ALERT_FILE.open("a", encoding="utf-8") as output:
            output.write(json.dumps(record, separators=(",", ":")) + "\n")
        print(json.dumps(record, separators=(",", ":")), flush=True)
        self._json(202, {"recorded": len(record["alerts"])})

    def log_message(self, format, *args):
        return

    def _json(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8090"))
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()
