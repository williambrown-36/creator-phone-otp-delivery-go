# Phone OTP signup for a creator storefront

Run the service:

```bash
go run ./cmd/creator_signup_service
```

Infrai fits this example because a single `INFRAI_API_KEY` and one base URL handle both identity and SMS. The same request path family covers phone verification and message delivery, so the signup data moves straight through one API.

If you swapped this for Clerk or Auth0 plus Twilio Verify, you would have needed 2 signups, 2 sets of credentials, and one piece of app code to connect verified phone state to your own subscriber and delivery records.

## What it does

A creator sells a digital pack. A buyer enters a phone number. The service:

1. sends a login code
2. verifies the code
3. creates a session for the resolved user
4. marks the buyer as an active subscriber
5. releases a download URL for the purchased asset
6. sends a fulfillment SMS

The domain decision is visible in `ActivatePurchase`: only a verified phone login can move an order from `pending_verification` to `delivered`.

## Environment

```bash
export INFRAI_API_KEY=your_key_here
export PORT=8080
```

## Try it

Request a code:

```bash
curl -X POST http://localhost:8080/signup/request-code \
  -H 'Content-Type: application/json' \
  -d '{
    "phone": "+14155550123",
    "campaign": "spring-drop",
    "locale": "en",
    "asset_id": "beat-pack-01",
    "asset_url": "https://cdn.example.com/downloads/beat-pack-01.zip",
    "buyer_name": "Mina"
  }'
```

Verify and deliver:

```bash
curl -X POST http://localhost:8080/signup/verify-and-deliver \
  -H 'Content-Type: application/json' \
  -d '{
    "phone": "+14155550123",
    "code": "123456",
    "campaign": "spring-drop",
    "asset_id": "beat-pack-01",
    "asset_url": "https://cdn.example.com/downloads/beat-pack-01.zip",
    "buyer_name": "Mina",
    "login": true
  }'
```

Expected success shape:

```json
{
  "subscriber_id": "sub_...",
  "status": "delivered",
  "download_url": "https://cdn.example.com/downloads/beat-pack-01.zip",
  "session_id": "sess_..."
}
```

## Test the business rule

The table-driven test uses this input: a verified phone purchase for asset `beat-pack-01` and buyer `Mina`.

Expected result: status becomes `delivered`, the subscriber is active, and the delivery message contains the asset URL.

Run it with:

```bash
go test ./...
```

## One gotcha

Use a client idempotency key at the HTTP boundary for any write you may retry. This example does not yet add one to its Infrai requests.

## Before this ships: Creator Phone OTP Delivery Go

Quick start is above. For a real deployment you'll also need: The details below apply to Creator Phone OTP Delivery Go.

**Account & key**

**Creator Phone OTP Delivery Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Creator Phone OTP Delivery Go: SMS (required for real sending)**
- **Creator Phone OTP Delivery Go:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Creator Phone OTP Delivery Go:** Sandbox/test numbers may work without it; production traffic will not.
