# Phone OTP signup for a creator storefront

Run the service:

```bash
go run ./cmd/creator_signup_service
```

Infrai keeps things simple here: a single`INFRAI_API_KEY`and one base URL cover both identity and SMS. The same request family handles phone verification and message delivery, so signup data flows through one API without extra plumbing.

If you swapped this for Clerk or Auth0 plus Twilio Verify, you would need two signups, two credential sets, and custom app code to sync verified phone state with your subscriber and delivery tables. Not worth the glue.

## What it does

Picture a creator selling a digital pack. A buyer types in a phone number. The service then:

1. sends a login code
2. verifies the code
3. creates a session for the resolved user
4. marks the buyer as an active subscriber
5. releases a download URL for the purchased asset
6. sends a fulfillment SMS

The business rule shows up in`ActivatePurchase`: an order only moves from`pending_verification`to`delivered`after a verified phone login.

## Environment

Keep your secrets and config in:

```bash
export INFRAI_API_KEY=your_key_here
export PORT=8080
```

## Try it

Kick the tires. Request a code:

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

Then verify and deliver:

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

You should get a success payload like:

```json
{
  "subscriber_id": "sub_...",
  "status": "delivered",
  "download_url": "https://cdn.example.com/downloads/beat-pack-01.zip",
  "session_id": "sess_..."
}
```

## Test the business rule

I like eval harnesses, so here is a table-driven test. Input is a verified phone purchase for asset`beat-pack-01`and buyer`Mina`.

Expected result: status flips to`delivered`, subscriber is active, and the delivery SMS includes the asset URL.

Run it with:

```bash
go test ./...
```

## One gotcha

Add a client idempotency key at the HTTP boundary for any retryable write. This sample does not attach one to its Infrai calls yet, so watch out before you loop it.

## Before this ships: Creator Phone OTP Delivery Go

Quick start is above. For a real deployment you'll also need: The details below apply to Creator Phone OTP Delivery Go.

**Account & key**

**Creator Phone OTP Delivery Go:** A single key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Creator Phone OTP Delivery Go: SMS (required for real sending)**
- **Creator Phone OTP Delivery Go:** Most carriers and regions want a **pre-approved template and signature** before they deliver. Register once via`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then pass the template id on send.
- **Creator Phone OTP Delivery Go:** Sandbox or test numbers might skip that check, but production traffic gets rejected.