import os
import random
import secrets
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, List, Optional

from sql_common.crypto_utils import hash_flag, hash_password

UTC = timezone.utc
DEFAULT_STACK_TARGET_PORTS = [{"container_port": 80, "protocol": "TCP"}]


def _render_username(pattern: str, team_name: str, number: int) -> str:
    return pattern.replace("{team_name}", team_name).replace("{number}", str(number))


def generate_teams(team_specs: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    base_time = datetime.now(UTC)
    teams: List[Dict[str, Any]] = []

    for idx, team in enumerate(team_specs, start=1):
        created_at = base_time + timedelta(minutes=idx)
        teams.append(
            {
                "id": idx,
                "name": team["name"],
                "created_at": created_at.strftime("%Y-%m-%d %H:%M:%S"),
            }
        )

    return teams


def generate_users(
    team_specs: List[Dict[str, Any]],
    bcrypt_cost: int,
    base_time: Optional[datetime] = None,
) -> List[Dict[str, Any]]:
    users: List[Dict[str, Any]] = []
    base_time = base_time or datetime.now(UTC)
    user_id = 1

    for team_idx, team in enumerate(team_specs, start=1):
        users_config = team.get("users") or {}
        if not users_config.get("enabled", False):
            continue

        count = int(users_config["count"])
        pattern = str(users_config["name_pattern"])
        emails = list(users_config["emails"])
        numbers = list(range(1, count + 1))
        random.shuffle(numbers)

        for idx in range(count):
            number = numbers[idx]
            username = _render_username(pattern, team["name"], number)
            email = emails[idx]
            plaintext = secrets.token_hex(32)
            password_hash = hash_password(plaintext, bcrypt_cost)
            created_at = base_time + timedelta(minutes=user_id)
            created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
            users.append(
                {
                    "id": user_id,
                    "email": email,
                    "username": username,
                    "password_hash": password_hash,
                    "role": "user",
                    "team_id": team_idx,
                    "created_at": created_at_str,
                    "updated_at": created_at_str,
                    "plaintext_password": plaintext,
                }
            )
            user_id += 1

    return users


def generate_challenges(
    challenges: List[Dict[str, Any]],
    constraints: Dict[str, Any],
    bcrypt_cost: int,
) -> List[Dict[str, Any]]:
    generated: List[Dict[str, Any]] = []
    base_time = datetime.now(UTC)
    ratio = constraints["min_points_ratio"]
    floor = constraints["min_points_floor"]

    for idx, chal in enumerate(challenges, start=1):
        created_at = base_time + timedelta(minutes=idx)
        created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
        points = int(chal["points"])
        minimum_points = max(floor, int(points * ratio))
        flag_hash = hash_flag(chal["flag"], bcrypt_cost)

        stack_enabled = bool(chal.get("stack_enabled", False))
        stack_target_ports = []
        stack_pod_spec = str(chal.get("stack_pod_spec", ""))
        if stack_enabled:
            stack_target_ports = list(chal.get("stack_target_ports", []))
            if not stack_target_ports:
                stack_target_ports = list(DEFAULT_STACK_TARGET_PORTS)
            if not stack_pod_spec:
                raise SystemExit(
                    f"Error: stack_enabled challenge {chal.get('title')} requires stack_pod_spec"
                )

        file_name = chal.get("file_name")
        file_key = chal.get("file_key")
        file_uploaded_at = chal.get("file_uploaded_at")
        if file_name and not file_key:
            file_key = f"{uuid.uuid4()}.zip"
        if file_name and not file_uploaded_at:
            file_uploaded_at = created_at_str

        generated.append(
            {
                "id": idx,
                "title": chal["title"],
                "description": chal["description"],
                "category": chal["category"],
                "points": points,
                "minimum_points": minimum_points,
                "flag_hash": flag_hash,
                "previous_challenge_id": chal.get("previous_challenge_id"),
                "is_active": bool(chal.get("is_active", True)),
                "created_at": created_at_str,
                "stack_enabled": stack_enabled,
                "stack_target_ports": stack_target_ports,
                "stack_pod_spec": stack_pod_spec,
                "file_key": file_key,
                "file_name": file_name,
                "file_uploaded_at": file_uploaded_at,
            }
        )

    return generated


def apply_challenge_pod_spec_paths(challenges: List[Dict[str, Any]], base_dir: str) -> None:
    from sql_common.yaml_utils import resolve_path

    for chal in challenges:
        pod_spec_path = chal.get("stack_pod_spec_path")
        if not pod_spec_path:
            continue
        resolved = resolve_path(pod_spec_path, base_dir)
        if not os.path.exists(resolved):
            raise SystemExit(
                f"Error: challenge pod spec file not found: {pod_spec_path}"
            )
        with open(resolved, "r", encoding="utf-8") as f:
            chal["stack_pod_spec"] = f.read().rstrip("\n")
