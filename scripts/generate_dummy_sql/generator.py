import hashlib
import hmac
import random
from datetime import datetime, timedelta, timezone
from typing import Any, Dict, List, Optional, Tuple

try:
    import bcrypt
except ImportError:
    bcrypt = None

UTC = timezone.utc


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
    team_ids: List[int],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    auth: Dict[str, Any],
    bcrypt_cost: int,
) -> List[Tuple[str, str, str, str, str, Optional[int]]]:
    users = []
    selected_names = random.sample(user_names, count - 1)

    base_time = datetime.now(UTC) - timedelta(hours=timing["users_base_hours_ago"])

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
            None,
        )
    )

    team_join_chance = probabilities["user"]["team_join_chance"]
    spread_hours = timing["user_created_hours_spread"]

    for korean_name in selected_names:
        username = korean_name["username"]
        email = f"{username}@example.com"
        password_hash = hash_password(auth["default_password"], bcrypt_cost)
        created_at = base_time + timedelta(hours=random.random() * spread_hours)
        created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
        team_id = None
        if team_ids and random.random() < team_join_chance:
            team_id = random.choice(team_ids)

        users.append((email, username, password_hash, "user", created_at_str, team_id))

    return users


def generate_challenges(
    challenges: List[Dict[str, Any]],
    timing: Dict[str, Any],
    constraints: Dict[str, Any],
    secret: str,
) -> List[Tuple[str, str, str, int, int, str, bool, str]]:
    generated = []
    base_time = datetime.now(UTC) - timedelta(hours=timing["challenges_base_hours_ago"])
    step_minutes = timing["challenge_created_minutes_step"]
    ratio = constraints["min_points_ratio"]
    floor = constraints["min_points_floor"]

    for i, chal in enumerate(challenges):
        flag_hash = hmac_flag(secret, chal["flag"])
        minimum_points = max(floor, int(chal["points"] * ratio))
        created_at = base_time + timedelta(minutes=i * step_minutes)
        generated.append(
            (
                chal["title"],
                chal["description"],
                chal["category"],
                chal["points"],
                minimum_points,
                flag_hash,
                True,
                created_at.strftime("%Y-%m-%d %H:%M:%S"),
            )
        )

    return generated


def generate_registration_keys(
    user_count: int,
    team_ids: List[int],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    count: int,
) -> List[
    Tuple[str, int, Optional[int], Optional[int], Optional[str], str, Optional[str]]
]:
    keys = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing["registration_keys_base_hours_ago"]
    )
    step_minutes = timing["registration_key_minutes_step"]
    used_limit = max(
        1, int(count * probabilities["registration_keys"]["used_fraction"])
    )
    seen_codes = set()

    team_assign_chance = probabilities["registration_keys"]["team_assign_chance"]

    for i in range(count):
        code = f"{random.randint(0, 999999):06d}"
        while code in seen_codes:
            code = f"{random.randint(0, 999999):06d}"
        seen_codes.add(code)

        created_at = base_time + timedelta(minutes=i * step_minutes)
        created_at_str = created_at.strftime("%Y-%m-%d %H:%M:%S")
        used_by = None
        used_by_ip = None
        used_at_str = None
        team_id = None

        if team_ids and random.random() < team_assign_chance:
            team_id = random.choice(team_ids)

        if i < used_limit and user_count > 1:
            used_by = random.randint(2, user_count)
            used_by_ip = f"203.0.113.{random.randint(1, 254)}"
            used_at = created_at + timedelta(minutes=random.randint(5, 180))
            used_at_str = used_at.strftime("%Y-%m-%d %H:%M:%S")

        keys.append(
            (code, 1, team_id, used_by, used_by_ip, created_at_str, used_at_str)
        )

    return keys


def generate_submissions(
    users: List[Tuple[str, str, str, str, str, Optional[int]]],
    challenges: List[Dict[str, Any]],
    timing: Dict[str, Any],
    probabilities: Dict[str, Any],
    secret: str,
) -> List[Tuple[int, int, str, bool, str]]:
    submissions = []
    base_time = datetime.now(UTC) - timedelta(
        hours=timing["submissions_base_hours_ago"]
    )

    user_team_map = {idx + 1: user[5] for idx, user in enumerate(users)}
    team_solved = {
        team_id: set() for team_id in set(user_team_map.values()) if team_id is not None
    }

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

    for user_id in range(2, len(users) + 1):
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

            if (
                unique_team_solve
                and team_id is not None
                and chal_id in team_solved.get(team_id, set())
            ):
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
                if unique_team_solve and team_id is not None:
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
    return submissions
