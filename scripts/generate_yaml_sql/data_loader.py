from typing import Any, Dict, List

from sql_common.yaml_utils import load_yaml

REQUIRED_CHALLENGE_KEYS = {"title", "description", "points", "flag", "category"}


def load_data(path: str) -> Dict[str, Any]:
    data = load_yaml(path)
    if "teams" not in data or "challenges" not in data:
        raise SystemExit("Error: data YAML must include teams and challenges")
    return data


def _validate_team_spec(team: Dict[str, Any], idx: int) -> None:
    if not isinstance(team, dict):
        raise SystemExit(f"Error: team entry {idx} must be a mapping")
    name = team.get("name")
    if not isinstance(name, str) or not name.strip():
        raise SystemExit(f"Error: team entry {idx} must include a non-empty name")

    users = team.get("users")
    if users is None:
        return
    if not isinstance(users, dict):
        raise SystemExit(f"Error: team entry {idx} users must be a mapping")

    enabled = bool(users.get("enabled", False))
    if not enabled:
        return

    if "count" not in users or "name_pattern" not in users or "emails" not in users:
        raise SystemExit(
            f"Error: team entry {idx} users requires count, name_pattern, and emails when enabled"
        )

    count = users.get("count")
    if not isinstance(count, int) or count <= 0:
        raise SystemExit(
            f"Error: team entry {idx} users count must be a positive integer"
        )

    name_pattern = users.get("name_pattern")
    if not isinstance(name_pattern, str) or not name_pattern.strip():
        raise SystemExit(
            f"Error: team entry {idx} users name_pattern must be a non-empty string"
        )

    emails = users.get("emails")
    if not isinstance(emails, list) or not emails:
        raise SystemExit(
            f"Error: team entry {idx} users emails must be a non-empty list"
        )
    if len(emails) != count:
        raise SystemExit(
            f"Error: team entry {idx} users emails length must match count ({count})"
        )
    for email_idx, email in enumerate(emails, start=1):
        if not isinstance(email, str) or not email.strip():
            raise SystemExit(
                f"Error: team entry {idx} users email {email_idx} must be a non-empty string"
            )


def validate_data(data: Dict[str, Any]) -> None:
    divisions = data.get("divisions", [])
    teams = data.get("teams", [])
    challenges = data.get("challenges", [])

    if not isinstance(divisions, list) or not divisions:
        raise SystemExit("Error: divisions must be a non-empty list")
    division_names: List[str] = []
    for idx, division in enumerate(divisions, start=1):
        if not isinstance(division, str) or not division.strip():
            raise SystemExit(f"Error: division entry {idx} must be a non-empty string")
        normalized = division.strip()
        if normalized.lower() == "admin":
            raise SystemExit("Error: division name 'Admin' is reserved for bootstrap")
        division_names.append(normalized)
    if len(division_names) != len(set(division_names)):
        raise SystemExit("Error: divisions must be unique")

    if not isinstance(teams, list) or not teams:
        raise SystemExit("Error: teams must be a non-empty list")
    if not isinstance(challenges, list) or not challenges:
        raise SystemExit("Error: challenges must be a non-empty list")

    team_names = []
    for idx, team in enumerate(teams, start=1):
        _validate_team_spec(team, idx)
        division = team.get("division")
        if division is not None:
            if not isinstance(division, str) or not division.strip():
                raise SystemExit(
                    f"Error: team entry {idx} division must be a non-empty string when provided"
                )
            normalized_division = division.strip()
            if normalized_division not in division_names:
                raise SystemExit(
                    f"Error: team entry {idx} division '{division}' not found in divisions"
                )
        team_names.append(team.get("name"))

    if len(team_names) != len(set(team_names)):
        raise SystemExit("Error: team names must be unique")

    for idx, chal in enumerate(challenges, start=1):
        if not isinstance(chal, dict) or not REQUIRED_CHALLENGE_KEYS.issubset(
            chal.keys()
        ):
            raise SystemExit(f"Error: challenge entry {idx} is missing required fields")

        points = chal.get("points")
        if not isinstance(points, int) or points < 0:
            raise SystemExit(
                f"Error: challenge entry {idx} points must be a non-negative integer"
            )
