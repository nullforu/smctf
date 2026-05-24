import os
import random
import secrets
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, List, Optional

from sql_common.crypto_utils import hash_flag, hash_password

UTC = timezone.utc
DEFAULT_VM_TARGET_PORTS = [{"container_port": 80, "protocol": "TCP"}]


def _render_username(pattern: str, team_name: str, number: int) -> str:
    return pattern.replace("{team_name}", team_name).replace("{number}", str(number))


def generate_divisions(
    division_names: List[str], id_offset: int = 0
) -> List[Dict[str, Any]]:
    base_time = datetime.now(UTC)
    divisions: List[Dict[str, Any]] = []

    for idx, name in enumerate(division_names, start=1):
        name = name.strip()
        division_id = idx + id_offset
        created_at = base_time + timedelta(minutes=division_id)
        divisions.append(
            {
                "id": division_id,
                "name": name,
                "created_at": created_at.strftime("%Y-%m-%d %H:%M:%S"),
            }
        )

    return divisions


def generate_teams(
    team_specs: List[Dict[str, Any]],
    division_map: Dict[str, int],
    default_division: str,
    id_offset: int = 0,
) -> List[Dict[str, Any]]:
    base_time = datetime.now(UTC)
    teams: List[Dict[str, Any]] = []

    for idx, team in enumerate(team_specs, start=1):
        team_id = idx + id_offset
        created_at = base_time + timedelta(minutes=team_id)
        division_name = team.get("division") or default_division
        division_name = division_name.strip()
        division_id = division_map.get(division_name)
        if division_id is None:
            raise SystemExit(f"Error: team division '{division_name}' not found")
        teams.append(
            {
                "id": team_id,
                "name": team["name"],
                "division_id": division_id,
                "division_name": division_name,
                "created_at": created_at.strftime("%Y-%m-%d %H:%M:%S"),
            }
        )

    return teams


def generate_users(
    team_specs: List[Dict[str, Any]],
    bcrypt_cost: int,
    base_time: Optional[datetime] = None,
    id_offset: int = 0,
    team_id_offset: int = 0,
) -> List[Dict[str, Any]]:
    users: List[Dict[str, Any]] = []
    base_time = base_time or datetime.now(UTC)
    user_id = 1 + id_offset

    for team_idx, team in enumerate(team_specs, start=1):
        team_id = team_idx + team_id_offset
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
                    "team_id": team_id,
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

        vm_enabled = bool(chal.get("vm_enabled", False))
        vm_spec = str(chal.get("vm_spec", "")) if vm_enabled else ""
        if vm_enabled and not vm_spec:
            raise SystemExit(
                f"Error: vm_enabled challenge {chal.get('title')} requires vm_spec"
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
                "vm_enabled": vm_enabled,
                "vm_spec": vm_spec,
                "file_key": file_key,
                "file_name": file_name,
                "file_uploaded_at": file_uploaded_at,
            }
        )

    return generated


def apply_challenge_vm_spec_paths(challenges: List[Dict[str, Any]], base_dir: str) -> None:
    from sql_common.yaml_utils import resolve_path

    for chal in challenges:
        vm_spec_path = chal.get("vm_spec_path")
        if not vm_spec_path:
            continue
        resolved = resolve_path(vm_spec_path, base_dir)
        if not os.path.exists(resolved):
            raise SystemExit(
                f"Error: challenge sandbox spec file not found: {vm_spec_path}"
            )
        with open(resolved, "r", encoding="utf-8") as f:
            chal["vm_spec"] = f.read().rstrip("\n")
