import hashlib
import hmac
import random
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, Iterable, List, Optional, Set, Tuple

try:
    import bcrypt
except ImportError:
    bcrypt = None

UTC = timezone.utc
REGISTRATION_CODE_ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
REGISTRATION_CODE_LENGTH = 16
REGISTRATION_CODE_MAX_USES = 3


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


def generate_teams(
    team_names: List[str], timing: Dict[str, Any]
) -> List[Tuple[str, str]]:
    teams = []
    base_time = datetime.now(UTC) - timedelta(hours=timing["teams_base_hours_ago"])
    step_minutes = timing["team_created_minutes_step"]

    for i, name in enumerate(team_names):
        created_at = base_time + timedelta(minutes=i * step_minutes)
        teams.append((name, created_at.strftime("%Y-%m-%d %H:%M:%S")))

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
    secret: str,
    stack_config: Optional[Dict[str, Any]] = None,
    stack_pod_spec_content: str = "",
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
        List[Dict[str, Any]],
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
    stack_config = stack_config or {}
    file_config = file_config or {}
    stack_enabled_default = bool(stack_config.get("enabled", False))
    stack_random_count = int(stack_config.get("random_challenge_count", 0))
    stack_target_ports_default = stack_config.get("target_ports")
    if not stack_target_ports_default and "target_port" in stack_config:
        stack_target_ports_default = [{"container_port": int(stack_config["target_port"]), "protocol": "TCP"}]
    if not stack_target_ports_default:
        stack_target_ports_default = [{"container_port": 80, "protocol": "TCP"}]
    file_enabled_default = bool(file_config.get("enabled", False))
    file_random_count = int(file_config.get("random_challenge_count", 0))
    file_default_name = str(file_config.get("file_name", "challenge.zip"))
    file_uploaded_after_max = int(file_config.get("uploaded_minutes_after_create_max", 120))
    stack_indices = _pick_random_indices(
        len(challenges), stack_random_count if stack_enabled_default else 0
    )
    file_indices = _pick_random_indices(
        len(challenges),
        file_random_count if file_enabled_default else 0,
        exclude=set(),
    )

    for i, chal in enumerate(challenges):
        flag_hash = hmac_flag(secret, chal["flag"])
        minimum_points = max(floor, int(chal["points"] * ratio))
        created_at = base_time + timedelta(minutes=i * step_minutes)
        stack_enabled = bool(chal.get("stack_enabled", False))
        if not stack_enabled and i in stack_indices:
            stack_enabled = True
        stack_target_ports = list(chal.get("stack_target_ports", [])) if stack_enabled else []
        if stack_enabled and not stack_target_ports:
            stack_target_ports = list(stack_target_ports_default)
        stack_pod_spec = str(chal.get("stack_pod_spec", "")) if stack_enabled else ""
        if stack_enabled and not stack_pod_spec:
            stack_pod_spec = stack_pod_spec_content
        if stack_enabled and not stack_pod_spec:
            raise SystemExit(
                "Error: stack_enabled challenge requires stack_pod_spec content"
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
                stack_enabled,
                stack_target_ports,
                stack_pod_spec,
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
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    secret: str,
    start_user_id: int = 1,
    skip_first_user: bool = False,
) -> List[Tuple[int, int, str, bool, str, bool]]:
    submissions = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing["submissions_base_hours_ago"]
    )

    user_team_map = {start_user_id + idx: user[5] for idx, user in enumerate(users)}
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
                    wrong_flag = f"flag{{wrong_attempt_{random.randint(1000, 9999)}}}"
                    wrong_hash = hmac_flag(secret, wrong_flag)
                    submissions.append(
                        (
                            user_id,
                            chal_id,
                            wrong_hash,
                            False,
                            wrong_time.strftime("%Y-%m-%d %H:%M:%S"),
                        )
                    )

                correct_flag = challenges[chal_id - 1]["flag"]
                correct_hash = hmac_flag(secret, correct_flag)
                submissions.append(
                    (
                        user_id,
                        chal_id,
                        correct_hash,
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
                wrong_flag = f"flag{{incorrect_{random.randint(1000, 9999)}}}"
                wrong_hash = hmac_flag(secret, wrong_flag)
                submissions.append(
                    (
                        user_id,
                        chal_id,
                        wrong_hash,
                        False,
                        attempt_time.strftime("%Y-%m-%d %H:%M:%S"),
                    )
                )

    now = datetime.now(UTC)
    recent_count = max(1, int(len(submissions) * recent_fraction))
    recent_indices = random.sample(range(len(submissions)), recent_count)

    for idx in recent_indices:
        recent_time = now - timedelta(minutes=random.randint(0, recent_minutes))
        user_id, chal_id, provided, correct, _ = submissions[idx]
        submissions[idx] = (
            user_id,
            chal_id,
            provided,
            correct,
            recent_time.strftime("%Y-%m-%d %H:%M:%S"),
        )

    submissions.sort(key=lambda x: x[4])

    first_blood_seen = set()
    flagged = []
    for user_id, chal_id, provided, correct, submitted_at in submissions:
        is_first_blood = False
        if correct and chal_id not in first_blood_seen:
            is_first_blood = True
            first_blood_seen.add(chal_id)
        flagged.append(
            (
                user_id,
                chal_id,
                provided,
                correct,
                submitted_at,
                is_first_blood,
            )
        )

    return flagged
