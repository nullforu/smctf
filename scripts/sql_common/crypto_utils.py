import hashlib
import hmac

try:
    import bcrypt
except ImportError:  # pragma: no cover - runtime dependency
    bcrypt = None


def _ensure_bcrypt() -> None:
    if bcrypt is None:
        raise SystemExit(
            "Error: bcrypt is required. Install it with: pip install bcrypt"
        )


def hmac_flag(secret: str, flag: str) -> str:
    h = hmac.new(secret.encode(), flag.encode(), hashlib.sha256)
    return h.hexdigest()


def hash_password(password: str, cost: int) -> str:
    _ensure_bcrypt()
    hashed = bcrypt.hashpw(password.encode(), bcrypt.gensalt(rounds=cost))
    return hashed.decode()
