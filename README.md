# Ory Kratos Disposable Email Webhook

A webhook service for Ory Kratos Action that validates email addresses against a list of disposable email domains. This service helps prevent users from registering with temporary/disposable email addresses.

## Ory Kratos Integration

### Configure Kratos Webhook

```yaml
selfservice:
  flows:
    registration:
      after:
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://localhost:8080/v1/validate
                method: POST
                headers:
                  X-API-Key: your-secret-api-key-change-me
                body: base64://ZnVuY3Rpb24oY3R4KSBjdHguaWRlbnRpdHkudHJhaXRzLmVtYWls
                response:
                  ignore: true
                  parse: false
```

### Body Payload (JSONNET)

```jsonnet
function(ctx) {
  email: ctx.identity.traits.email,
}
```

## API Endpoints

### POST /v1/validate/email

Email validation endpoint for Ory Kratos.

**Headers**:
- `X-API-Key`: Your API key (required)
- `Content-Type`: application/json

**Request Body**:
```json
{ "email": "user@example.com" }
```

**Success Response** (HTTP 200):
Email is valid, flow continues.

**Error Response** (HTTP 400):
```json
{
  "messages": [{
    "instance_ptr": "#/traits/email",
    "messages": [{
      "id": 4000001,
      "text": "Disposable email addresses are not allowed",
      "type": "error",
      "context": {
        "email": "user@tempmail.com",
        "domain": "tempmail.com"
      }
    }]
  }]
}
```

### Webhook Behavior

1. **Before Registration**: User submits registration form with email
2. **Webhook Called**: Kratos sends email to this webhook for validation
3. **Validation**: Service checks if email domain is disposable
4. **Response**:
   - If valid → HTTP 200 → Registration continues
   - If disposable → HTTP 400 with error → Registration blocked with error message
