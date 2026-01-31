from typing import Any, Dict

from config_loader import load_yaml


def load_data(path: str) -> Dict[str, Any]:
    data = load_yaml(path)
    if "users" not in data or "teams" not in data or "challenges" not in data:
        raise SystemExit("Error: data YAML must include users, teams, and challenges")
    return data


def validate_data(data: Dict[str, Any], user_count: int, min_user_names: int) -> None:
    users = data.get("users", [])
    teams = data.get("teams", [])
    challenges = data.get("challenges", [])

    if not isinstance(users, list) or not users:
        raise SystemExit("Error: users must be a non-empty list")
    if not isinstance(teams, list) or not teams:
        raise SystemExit("Error: teams must be a non-empty list")
    if not isinstance(challenges, list) or not challenges:
        raise SystemExit("Error: challenges must be a non-empty list")

    if len(users) < min_user_names:
        raise SystemExit(
            f"Error: users must contain at least {min_user_names} entries (found {len(users)})"
        )

    required_user_keys = {"name", "username"}
    for idx, user in enumerate(users, start=1):
        if not isinstance(user, dict) or not required_user_keys.issubset(user.keys()):
            raise SystemExit(f"Error: user entry {idx} must contain name and username")

    required_challenge_keys = {"title", "description", "points", "flag", "category"}
    for idx, chal in enumerate(challenges, start=1):
        if not isinstance(chal, dict) or not required_challenge_keys.issubset(
            chal.keys()
        ):
            raise SystemExit(f"Error: challenge entry {idx} is missing required fields")

    if user_count - 1 > len(users):
        raise SystemExit(
            f"Error: requested {user_count} users but only {len(users)} user names available"
        )
