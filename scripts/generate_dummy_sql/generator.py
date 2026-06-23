import random
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, Iterable, List, Optional, Set, Tuple

from sql_common.crypto_utils import hash_flag, hash_password

UTC = timezone.utc
REGISTRATION_CODE_ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
REGISTRATION_CODE_LENGTH = 16
REGISTRATION_CODE_MAX_USES = 3


def _pick_random_indices(
    total: int, count: int, exclude: Optional[Iterable[int]] = None
) -> Set[int]:
    if total <= 0 or count <= 0:
        return set()
    excluded = set(exclude or [])
    candidates = [idx for idx in range(total) if idx not in excluded]
    if not candidates:
        return set()
    count = min(count, len(candidates))
    return set(random.sample(candidates, count))


def generate_divisions(
    division_names: List[str], timing: Dict[str, Any]
) -> List[Tuple[str, str]]:
    divisions = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing.get("divisions_base_hours_ago", timing["teams_base_hours_ago"])
    )
    step_minutes = timing.get("division_created_minutes_step", 5)

    for i, name in enumerate(division_names):
        created_at = base_time + timedelta(minutes=i * step_minutes)
        divisions.append((name, created_at.strftime("%Y-%m-%d %H:%M:%S")))

    return divisions


def generate_teams(
    team_specs: List[Dict[str, str]], timing: Dict[str, Any]
) -> List[Tuple[str, str, str]]:
    teams = []
    base_time = datetime.now(UTC) - timedelta(hours=timing["teams_base_hours_ago"])
    step_minutes = timing["team_created_minutes_step"]

    for i, team in enumerate(team_specs):
        name = team["name"]
        division = team["division"]
        created_at = base_time + timedelta(minutes=i * step_minutes)
        teams.append((name, division, created_at.strftime("%Y-%m-%d %H:%M:%S")))

    return teams


def generate_users(
    user_names: List[Dict[str, str]],
    count: int,
    admin_team_id: int,
    non_admin_team_ids: List[int],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    auth: Dict[str, Any],
    bcrypt_cost: int,
    include_admin: bool = True,
) -> List[Tuple[str, str, str, str, str, int]]:
    if include_admin and admin_team_id <= 0:
        raise ValueError("admin_team_id must be positive")
    if not non_admin_team_ids:
        raise ValueError("non_admin_team_ids must not be empty")

    users = []
    base_time = datetime.now(UTC) - timedelta(hours=timing["users_base_hours_ago"])

    if include_admin:
        admin = auth["admin"]
        admin_password_hash = hash_password(admin["password"], bcrypt_cost)
        admin_time = base_time.strftime("%Y-%m-%d %H:%M:%S")
        users.append(
            (
                admin["email"],
                admin["username"],
                admin_password_hash,
                admin["role"],
                admin_time,
                admin_team_id,
            )
        )

    remaining = count - (1 if include_admin else 0)
    if remaining < 0:
        remaining = 0
    selected_names = random.sample(user_names, remaining)

    spread_hours = timing["user_created_hours_spread"]

    for korean_name in selected_names:
        username = korean_name["username"]
        email = f"{username}@example.com"
        password_hash = hash_password(auth["default_password"], bcrypt_cost)
        created_at = base_time + timedelta(hours=random.random() * spread_hours)
        created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
        team_id = random.choice(non_admin_team_ids)

        users.append((email, username, password_hash, "user", created_at_str, team_id))

    return users


def generate_challenges(
    challenges: List[Dict[str, Any]],
    timing: Dict[str, Any],
    constraints: Dict[str, Any],
    bcrypt_cost: int,
    vm_config: Optional[Dict[str, Any]] = None,
    vm_spec_content: str = "",
    file_config: Optional[Dict[str, Any]] = None,
) -> List[
    Tuple[
        str,
        str,
        str,
        int,
        int,
        str,
        Optional[int],
        bool,
        str,
        bool,
        str,
        Optional[str],
        Optional[str],
        Optional[str],
    ]
]:
    generated = []
    base_time = datetime.now(UTC) - timedelta(hours=timing["challenges_base_hours_ago"])
    step_minutes = timing["challenge_created_minutes_step"]
    ratio = constraints["min_points_ratio"]
    floor = constraints["min_points_floor"]
    vm_config = vm_config or {}
    file_config = file_config or {}
    vm_enabled_default = bool(vm_config.get("enabled", False))
    vm_random_count = int(vm_config.get("random_challenge_count", 0))
    file_enabled_default = bool(file_config.get("enabled", False))
    file_random_count = int(file_config.get("random_challenge_count", 0))
    file_default_name = str(file_config.get("file_name", "challenge.zip"))
    file_uploaded_after_max = int(file_config.get("uploaded_minutes_after_create_max", 120))
    vm_indices = _pick_random_indices(
        len(challenges), vm_random_count if vm_enabled_default else 0
    )
    file_indices = _pick_random_indices(
        len(challenges),
        file_random_count if file_enabled_default else 0,
        exclude=set(),
    )

    for i, chal in enumerate(challenges):
        flag_hash = hash_flag(chal["flag"], bcrypt_cost)
        minimum_points = max(floor, int(chal["points"] * ratio))
        created_at = base_time + timedelta(minutes=i * step_minutes)
        vm_enabled = bool(chal.get("vm_enabled", False))
        if not vm_enabled and i in vm_indices:
            vm_enabled = True
        vm_spec = str(chal.get("vm_spec", "")) if vm_enabled else ""
        if vm_enabled and not vm_spec:
            vm_spec = vm_spec_content
        if vm_enabled and not vm_spec:
            raise SystemExit(
                "Error: vm_enabled challenge requires vm_spec content"
            )
        file_key = chal.get("file_key")
        file_name = chal.get("file_name")
        file_uploaded_at = chal.get("file_uploaded_at")
        if not file_name and i in file_indices:
            file_name = file_default_name
        if file_key is None and file_name:
            file_key = f"{uuid.UUID(int=random.getrandbits(128))}.zip"
        if file_uploaded_at is None and file_name:
            offset = 0 if file_uploaded_after_max <= 0 else random.randint(0, file_uploaded_after_max)
            uploaded_at = created_at + timedelta(minutes=offset)
            file_uploaded_at = uploaded_at.strftime("%Y-%m-%d %H:%M:%S")

        generated.append(
            (
                chal["title"],
                chal["description"],
                chal["category"],
                chal["points"],
                minimum_points,
                flag_hash,
                chal.get("previous_challenge_id"),
                True,
                created_at.strftime("%Y-%m-%d %H:%M:%S"),
                vm_enabled,
                vm_spec,
                file_key,
                file_name,
                file_uploaded_at,
            )
        )

    return generated


def generate_registration_keys(
    user_ids: List[int],
    team_ids: List[int],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    count: int,
    created_by: int,
) -> Tuple[
    List[Tuple[int, str, int, int, int, int, str]],
    List[Tuple[int, int, str, str]],
]:
    if not team_ids:
        raise ValueError("team_ids must not be empty")

    keys = []
    uses = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing["registration_keys_base_hours_ago"]
    )
    step_minutes = timing["registration_key_minutes_step"]
    used_limit = max(
        1, int(count * probabilities["registration_keys"]["used_fraction"])
    )
    candidate_users = [uid for uid in user_ids if uid != created_by]
    seen_codes = set()

    for i in range(count):
        code = "".join(
            random.choice(REGISTRATION_CODE_ALPHABET)
            for _ in range(REGISTRATION_CODE_LENGTH)
        )
        while code in seen_codes:
            code = "".join(
                random.choice(REGISTRATION_CODE_ALPHABET)
                for _ in range(REGISTRATION_CODE_LENGTH)
            )
        seen_codes.add(code)

        created_at = base_time + timedelta(minutes=i * step_minutes)
        created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
        key_id = i + 1
        team_id = random.choice(team_ids)
        max_uses = random.randint(1, REGISTRATION_CODE_MAX_USES)
        used_count = 0

        if i < used_limit and candidate_users:
            used_count = random.randint(1, max_uses)
            for _ in range(used_count):
                used_by = random.choice(candidate_users)
                used_by_ip = f"203.0.113.{random.randint(1, 254)}"
                used_at = created_at + timedelta(minutes=random.randint(5, 180))
                used_at_str = used_at.strftime("%Y-%m-%d %H:%M:%S")
                uses.append((key_id, used_by, used_by_ip, used_at_str))

        keys.append(
            (key_id, code, created_by, team_id, max_uses, used_count, created_at_str)
        )

    return keys, uses


def generate_submissions(
    users: List[Tuple[str, str, str, str, str, Optional[int]]],
    challenges: List[Dict[str, Any]],
    team_division_map: Optional[Dict[int, str]],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    start_user_id: int = 1,
    skip_first_user: bool = False,
) -> List[Tuple[int, int, bool, str, bool]]:
    submissions = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing["submissions_base_hours_ago"]
    )

    user_team_map = {start_user_id + idx: user[5] for idx, user in enumerate(users)}
    team_division_map = team_division_map or {}
    user_division_map: Dict[int, str] = {}
    for user_id, team_id in user_team_map.items():
        if team_id is None:
            raise SystemExit(f"Error: user id {user_id} has no team_id")
        division = team_division_map.get(team_id)
        if not isinstance(division, str) or not division.strip():
            raise SystemExit(
                f"Error: team id {team_id} has no valid division for user id {user_id}"
            )
        user_division_map[user_id] = division
    team_solved = {team_id: set() for team_id in set(user_team_map.values())}

    prob = probabilities["submissions"]
    attempts_min = prob["attempt_count"]["min"]
    attempts_max = prob["attempt_count"]["max"]
    beta_alpha = prob["skill_beta"]["alpha"]
    beta_beta = prob["skill_beta"]["beta"]
    weight_min = prob["challenge_weight"]["min"]
    weight_bias = prob["challenge_weight"]["skill_bias"]
    solve_min = prob["solve_probability"]["min"]
    solve_bias = prob["solve_probability"]["skill_bias"]
    wrong_values = prob["wrong_attempts"]["values"]
    wrong_weights = prob["wrong_attempts"]["weights"]
    wrong_before_min = prob["wrong_attempts_time"]["min_minutes_before"]
    wrong_before_max = prob["wrong_attempts_time"]["max_minutes_before"]
    fail_delay_min = prob["failure_attempt_delay"]["min_minutes"]
    fail_delay_max = prob["failure_attempt_delay"]["max_minutes"]
    recent_fraction = prob["recent_submissions"]["fraction"]
    recent_minutes = prob["recent_submissions"]["max_minutes_ago"]
    unique_team_solve = prob["team_unique_solve"]

    challenge_count = len(challenges)

    user_ids = list(user_team_map.keys())
    if skip_first_user and user_ids:
        user_ids = [uid for uid in user_ids if uid != start_user_id]

    for user_id in user_ids:
        skill_level = random.betavariate(beta_alpha, beta_beta)
        attempt_count = random.randint(attempts_min, attempts_max)
        attempted_challenges = set()

        for _ in range(attempt_count):
            challenge_weights = []
            for chal_id in range(1, challenge_count + 1):
                difficulty = chal_id / challenge_count
                weight = max(weight_min, skill_level - difficulty + weight_bias)
                challenge_weights.append(weight)

            chal_id = random.choices(
                range(1, challenge_count + 1), weights=challenge_weights
            )[0]
            attempted_challenges.add(chal_id)

        for chal_id in attempted_challenges:
            difficulty = chal_id / challenge_count
            submission_time = base_time + timedelta(hours=random.random() * 42)

            solve_probability = max(solve_min, skill_level - difficulty + solve_bias)
            will_solve = random.random() < solve_probability
            team_id = user_team_map.get(user_id)

            if unique_team_solve and chal_id in team_solved.get(team_id, set()):
                will_solve = False

            if will_solve:
                wrong_attempts = random.choices(wrong_values, weights=wrong_weights)[0]
                for _ in range(wrong_attempts):
                    wrong_time = submission_time - timedelta(
                        minutes=random.randint(wrong_before_min, wrong_before_max)
                    )
                    submissions.append(
                        (
                            user_id,
                            chal_id,
                            False,
                            wrong_time.strftime("%Y-%m-%d %H:%M:%S"),
                        )
                    )

                submissions.append(
                    (
                        user_id,
                        chal_id,
                        True,
                        submission_time.strftime("%Y-%m-%d %H:%M:%S"),
                    )
                )
                if unique_team_solve:
                    team_solved.setdefault(team_id, set()).add(chal_id)
            else:
                attempt_time = submission_time + timedelta(
                    minutes=random.randint(fail_delay_min, fail_delay_max)
                )
                submissions.append(
                    (
                        user_id,
                        chal_id,
                        False,
                        attempt_time.strftime("%Y-%m-%d %H:%M:%S"),
                    )
                )

    now = datetime.now(UTC)
    if submissions:
        recent_count = max(1, int(len(submissions) * recent_fraction))
        recent_count = min(recent_count, len(submissions))
        recent_indices = random.sample(range(len(submissions)), recent_count)

        for idx in recent_indices:
            recent_time = now - timedelta(minutes=random.randint(0, recent_minutes))
            user_id, chal_id, correct, _ = submissions[idx]
            submissions[idx] = (
                user_id,
                chal_id,
                correct,
                recent_time.strftime("%Y-%m-%d %H:%M:%S"),
            )

    submissions.sort(key=lambda x: x[3])

    first_blood_seen = set()
    flagged = []
    for user_id, chal_id, correct, submitted_at in submissions:
        division = user_division_map.get(user_id, "Unknown")
        is_first_blood = False
        key = (division, chal_id)
        if correct and key not in first_blood_seen:
            is_first_blood = True
            first_blood_seen.add(key)
        flagged.append(
            (
                user_id,
                chal_id,
                correct,
                submitted_at,
                is_first_blood,
            )
        )

    return flagged
