import os
from typing import Any, Dict, Iterable, Optional

try:
    import yaml
except ImportError:  # pragma: no cover - runtime dependency
    yaml = None


def _ensure_yaml() -> None:
    if yaml is None:
        raise SystemExit(
            "Error: PyYAML is required. Install it with: pip install pyyaml"
        )


def load_yaml(path: str) -> Dict[str, Any]:
    _ensure_yaml()
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    if not isinstance(data, dict):
        raise SystemExit(f"Error: YAML root must be a mapping: {path}")
    return data


def deep_merge(base: Dict[str, Any], override: Dict[str, Any]) -> Dict[str, Any]:
    result = dict(base)
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value
    return result


def load_settings(
    default_path: str, template_paths: Iterable[str], user_path: Optional[str]
) -> Dict[str, Any]:
    settings = load_yaml(default_path)

    for path in template_paths:
        settings = deep_merge(settings, load_yaml(path))

    if user_path:
        settings = deep_merge(settings, load_yaml(user_path))

    return settings


def resolve_path(path: str, base_dir: str) -> str:
    if os.path.isabs(path):
        return path
    return os.path.join(base_dir, path)
