#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import time, requests, logging

BASE = "http://192.168.0.254/api/v15"

APP_ID      = "fr.freebox.terraform"
APP_NAME    = "Terraform"
APP_VERSION = "1.0.0"
DEVICE_NAME = "Terraform"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)

def _post(url, payload):
    r = requests.post(url, json=payload, timeout=10)
    r.raise_for_status()
    data = r.json()
    if not data.get("success", False):
        raise RuntimeError(f"POST {url} failed: {data}")
    return data


def _get(url):
    r = requests.get(url, timeout=10)
    r.raise_for_status()
    data = r.json()
    if not data.get("success", False):
        raise RuntimeError(f"GET {url} failed: {data}")
    return data


def request_app_token():
    url = f"{BASE}/login/authorize/"
    payload = {
        "app_id":       APP_ID,
        "app_name":     APP_NAME,
        "app_version":  APP_VERSION,
        "device_name":  DEVICE_NAME,
    }
    data = _post(url, payload)
    app_token = data["result"]["app_token"]
    track_id  = data["result"]["track_id"]
    logging.info("Authorization request sent. Go approve it on your Freebox.")
    return app_token, track_id


def wait_for_grant(track_id):
    url = f"{BASE}/login/authorize/{track_id}"
    while True:
        data = _get(url)
        status = data["result"]["status"]
        logging.info(f"Status: {status}")
        if status == "granted":
            return
        if status in ("denied", "timeout", "unknown"):
            raise RuntimeError(f"Authorization failed: {status}")
        time.sleep(2)

def main():
    app_token, track_id = request_app_token()
    logging.info(f"Got track_id={track_id}. Please approve on the Freebox screen.")
    wait_for_grant(track_id)
    logging.info(f"✅ Your app_token is: {app_token}")
    print("
=============================")
    print(" APP TOKEN:", app_token)
    print(" BASE URL:", BASE)
    print("=============================
")

if __name__ == "__main__":
    main()