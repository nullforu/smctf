from typing import Any, Dict, Iterable, Optional

from sql_common.yaml_utils import deep_merge, load_yaml, resolve_path

def load_settings(
    default_path: str, template_paths: Iterable[str], user_path: Optional[str]
) -> Dict[str, Any]:
    settings = load_yaml(default_path)

    for path in template_paths:
        settings = deep_merge(settings, load_yaml(path))

    if user_path:
        settings = deep_merge(settings, load_yaml(user_path))

    return settings
