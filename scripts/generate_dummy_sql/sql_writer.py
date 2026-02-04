from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple


def escape_sql_string(s: str) -> str:
    return s.replace("'", "''")


def write_sql_file(
    output_file: str,
    teams: List[Tuple[str, str]],
    users: List[Tuple[str, str, str, str, str, int]],
    challenges: List[Tuple[str, str, str, int, int, str, bool, str]],
    registration_keys: List[
        Tuple[str, int, int, Optional[int], Optional[str], str, Optional[str]]
    ],
    submissions: List[Tuple[int, int, str, bool, str]],
    meta: Dict[str, Any],
) -> None:
    with open(output_file, "w", encoding="utf-8") as f:
        f.write("-- smctf Dummy Data\n")
        f.write(f"-- Generated at: {datetime.now().isoformat()}\n")
        f.write(f"-- FLAG_HMAC_SECRET: {meta['flag_hmac_secret']}\n")
        f.write(f"-- BCRYPT_COST: {meta['bcrypt_cost']}\n")
        f.write(f"-- Default password for all users: {meta['default_password']}\n")
        f.write(
            f"-- Admin credentials: {meta['admin_email']} / {meta['admin_password']}\n\n"
        )

        f.write("-- App Config\n")
        f.write("INSERT INTO app_config (key, value, updated_at) VALUES ('title', 'Welcome to My CTF!', NOW()), ('description', 'this is a sample CTF description.', NOW());\n\n")

        f.write("-- Clear existing data\n")
        f.write(
            "TRUNCATE TABLE submissions, registration_keys, challenges, users, teams RESTART IDENTITY CASCADE;\n\n"
        )

        f.write("-- Insert teams\n")
        for name, created_at in teams:
            name_esc = escape_sql_string(name)
            f.write("INSERT INTO teams (name, created_at) VALUES ")
            f.write(f"('{name_esc}', '{created_at}');\n")
        f.write("\n")

        f.write("-- Insert users\n")
        for email, username, password_hash, role, created_at, team_id in users:
            email_esc = escape_sql_string(email)
            username_esc = escape_sql_string(username)
            password_hash_esc = escape_sql_string(password_hash)
            role_esc = escape_sql_string(role)

            f.write(
                "INSERT INTO users (email, username, password_hash, role, team_id, created_at, updated_at) VALUES "
            )
            f.write(
                f"('{email_esc}', '{username_esc}', '{password_hash_esc}', '{role_esc}', {team_id}, '{created_at}', '{created_at}');\n"
            )

        f.write("\n")

        f.write("-- Insert registration keys\n")
        for (
            code,
            created_by,
            team_id,
            used_by,
            used_by_ip,
            created_at,
            used_at,
        ) in registration_keys:
            code_esc = escape_sql_string(code)
            used_by_value = "NULL" if used_by is None else str(used_by)
            used_at_value = "NULL" if used_at is None else f"'{used_at}'"
            used_by_ip_value = (
                "NULL" if used_by_ip is None else f"'{escape_sql_string(used_by_ip)}'"
            )

            f.write(
                "INSERT INTO registration_keys (code, created_by, team_id, used_by, used_by_ip, created_at, used_at) VALUES "
            )
            f.write(
                f"('{code_esc}', {created_by}, {team_id}, {used_by_value}, {used_by_ip_value}, '{created_at}', {used_at_value});\n"
            )

        f.write("\n")

        f.write("-- Insert challenges\n")
        for (
            title,
            description,
            category,
            points,
            minimum_points,
            flag_hash,
            is_active,
            created_at,
        ) in challenges:
            title_esc = escape_sql_string(title)
            description_esc = escape_sql_string(description)
            category_esc = escape_sql_string(category)
            flag_hash_esc = escape_sql_string(flag_hash)

            f.write(
                "INSERT INTO challenges (title, description, category, points, minimum_points, flag_hash, is_active, created_at) VALUES "
            )
            f.write(
                f"('{title_esc}', '{description_esc}', '{category_esc}', {points}, {minimum_points}, '{flag_hash_esc}', {is_active}, '{created_at}');\n"
            )

        f.write("\n")

        f.write("-- Insert submissions\n")
        for user_id, challenge_id, provided, correct, submitted_at in submissions:
            provided_esc = escape_sql_string(provided)

            f.write(
                "INSERT INTO submissions (user_id, challenge_id, provided, correct, submitted_at) VALUES "
            )
            f.write(
                f"({user_id}, {challenge_id}, '{provided_esc}', {correct}, '{submitted_at}');\n"
            )

        f.write("\n")
        f.write("-- Update sequences\n")
        f.write("SELECT setval('teams_id_seq', (SELECT MAX(id) FROM teams));\n")
        f.write("SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));\n")
        f.write(
            "SELECT setval('challenges_id_seq', (SELECT MAX(id) FROM challenges));\n"
        )
        f.write(
            "SELECT setval('registration_keys_id_seq', (SELECT MAX(id) FROM registration_keys));\n"
        )
        f.write(
            "SELECT setval('submissions_id_seq', (SELECT MAX(id) FROM submissions));\n"
        )
