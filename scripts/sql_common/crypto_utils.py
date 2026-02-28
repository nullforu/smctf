try:
    import bcrypt
except ImportError:  # pragma: no cover - runtime dependency
    bcrypt = None


def _ensure_bcrypt() -> None:
    if bcrypt is None:
        raise SystemExit(
            "Error: bcrypt is required. Install it with: pip install bcrypt"
        )


def hash_password(password: str, cost: int) -> str:
    _ensure_bcrypt()
    hashed = bcrypt.hashpw(password.encode(), bcrypt.gensalt(rounds=cost))
    return hashed.decode()


def hash_flag(flag: str, cost: int) -> str:
    return hash_password(flag, cost)
