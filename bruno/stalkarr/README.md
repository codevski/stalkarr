# Stalkarr — Bruno Collection

API tests for the Stalkarr backend. Uses [Bruno](https://www.usebruno.com/) a git-friendly API client.

## Setup

1. Install Bruno: https://www.usebruno.com/downloads
2. Open Bruno → **Open Collection** → select this `bruno/stalkarr` folder
3. Select the **Local** environment (top right dropdown)

## First run order

Run requests in this order when setting up a fresh instance:

1. `auth/Setup User` - creates the admin account (once only)
2. `auth/Login` - logs in and **automatically saves the token** to the environment
3. `settings/Save Settings` - add your Sonarr URL + API key
4. `sonarr/Get Missing Episodes` - verify the Sonarr connection works

## Notes

- The **Login** request auto-saves the JWT token via a post-response script — no copy/pasting needed
- API keys are never returned in plaintext `Get Settings` only shows the last 4 chars
- The `Local` environment `token` variable is populated automatically on login
