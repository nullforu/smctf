import json
from datetime import datetime
from typing import Any, Dict, List

from sql_common.sql_utils import escape_sql_string


def _sql_value(value: Any) -> str:
    if value is None:
        return "NULL"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    return f"'{escape_sql_string(str(value))}'"


def write_sql_file(
    output_file: str,
    teams: List[Dict[str, Any]],
    users: List[Dict[str, Any]],
    challenges: List[Dict[str, Any]],
    meta: Dict[str, Any],
) -> None:
    with open(output_file, "w", encoding="utf-8") as f:
        f.write("-- smctf YAML SQL\n")
        f.write(f"-- Generated at: {datetime.now().isoformat()}\n")
        f.write(f"-- BCRYPT_COST: {meta['bcrypt_cost']}\n\n")

        f.write("-- Guard: refuse to run if tables are not empty\n")
        f.write("DO $$\n")
        f.write("BEGIN\n")
        f.write(
            "  IF EXISTS (SELECT 1 FROM teams) OR EXISTS (SELECT 1 FROM challenges) OR EXISTS (SELECT 1 FROM users) THEN\n"
        )
        f.write("    RAISE EXCEPTION 'Refusing to run: tables not empty';\n")
        f.write("  END IF;\n")
        f.write("END $$;\n\n")

        f.write("-- Insert teams\n")
        for team in teams:
            f.write("INSERT INTO teams (id, name, created_at) VALUES ")
            f.write(
                f"({_sql_value(team['id'])}, {_sql_value(team['name'])}, {_sql_value(team['created_at'])});\n"
            )
        f.write("\n")

        if users:
            f.write("-- Insert users (with plaintext password comments)\n")
            for user in users:
                f.write(
                    f"-- User: {escape_sql_string(user['username'])} | Email: {escape_sql_string(user['email'])} | Password: {user['plaintext_password']}\n"
                )
                f.write(
                    "INSERT INTO users (id, email, username, password_hash, role, team_id, created_at, updated_at) VALUES "
                )
                f.write(
                    "({id}, {email}, {username}, {password_hash}, {role}, {team_id}, {created_at}, {updated_at});\n".format(
                        id=_sql_value(user["id"]),
                        email=_sql_value(user["email"]),
                        username=_sql_value(user["username"]),
                        password_hash=_sql_value(user["password_hash"]),
                        role=_sql_value(user["role"]),
                        team_id=_sql_value(user["team_id"]),
                        created_at=_sql_value(user["created_at"]),
                        updated_at=_sql_value(user["updated_at"]),
                    )
                )
            f.write("\n")

        f.write("-- Insert challenges\n")
        for chal in challenges:
            stack_target_ports_value = "NULL"
            if chal["stack_target_ports"]:
                ports_json = json.dumps(chal["stack_target_ports"], ensure_ascii=False)
                stack_target_ports_value = _sql_value(ports_json)

            f.write(
                "INSERT INTO challenges (id, title, description, category, points, minimum_points, flag_hash, previous_challenge_id, is_active, created_at, stack_enabled, stack_target_ports, stack_pod_spec, file_key, file_name, file_uploaded_at) VALUES "
            )
            f.write(
                "({id}, {title}, {description}, {category}, {points}, {minimum_points}, {flag_hash}, {previous_challenge_id}, {is_active}, {created_at}, {stack_enabled}, {stack_target_ports}, {stack_pod_spec}, {file_key}, {file_name}, {file_uploaded_at});\n".format(
                    id=_sql_value(chal["id"]),
                    title=_sql_value(chal["title"]),
                    description=_sql_value(chal["description"]),
                    category=_sql_value(chal["category"]),
                    points=_sql_value(chal["points"]),
                    minimum_points=_sql_value(chal["minimum_points"]),
                    flag_hash=_sql_value(chal["flag_hash"]),
                    previous_challenge_id=_sql_value(chal["previous_challenge_id"]),
                    is_active=_sql_value(chal["is_active"]),
                    created_at=_sql_value(chal["created_at"]),
                    stack_enabled=_sql_value(chal["stack_enabled"]),
                    stack_target_ports=stack_target_ports_value,
                    stack_pod_spec=_sql_value(chal["stack_pod_spec"] or None),
                    file_key=_sql_value(chal["file_key"]),
                    file_name=_sql_value(chal["file_name"]),
                    file_uploaded_at=_sql_value(chal["file_uploaded_at"]),
                )
            )
        f.write("\n")

        f.write("-- Update sequences\n")
        f.write("SELECT setval('teams_id_seq', (SELECT MAX(id) FROM teams));\n")
        if users:
            f.write("SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));\n")
        f.write(
            "SELECT setval('challenges_id_seq', (SELECT MAX(id) FROM challenges));\n"
        )
