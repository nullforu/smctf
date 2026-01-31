#!/usr/bin/env python3

import argparse
import os
import random
import sys
from typing import List

from config_loader import load_settings, resolve_path
from data_loader import load_data, validate_data
from generator import (
    generate_challenges,
    generate_registration_keys,
    generate_submissions,
    generate_teams,
    generate_users,
)
from sql_writer import write_sql_file

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DATA_PATH = os.path.join(BASE_DIR, "defaults", "data.yaml")
DEFAULT_SETTINGS_PATH = os.path.join(BASE_DIR, "defaults", "settings.yaml")
DEFAULT_TEMPLATES_DIR = os.path.join(BASE_DIR, "templates")


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate smctf dummy SQL data")
    parser.add_argument(
        "--data",
        help="Path to data YAML (users/teams/challenges). Defaults to bundled data.yaml.",
    )
    parser.add_argument(
        "--settings",
        help="Path to settings YAML (probabilities/timing). Merged over defaults.",
    )
    parser.add_argument(
        "--template",
        action="append",
        default=[],
        help="Template YAML to apply before settings (can be repeated).",
    )
    parser.add_argument(
        "--output",
        help="Override output SQL file path.",
    )
    parser.add_argument(
        "--seed",
        type=int,
        help="Random seed for reproducible output.",
    )
    parser.add_argument(
        "--list-templates",
        action="store_true",
        help="List bundled templates and exit.",
    )
    return parser.parse_args(argv)


def list_templates() -> None:
    if not os.path.isdir(DEFAULT_TEMPLATES_DIR):
        print("No templates directory found.")
        return
    templates = sorted(
        entry for entry in os.listdir(DEFAULT_TEMPLATES_DIR) if entry.endswith(".yaml")
    )
    if not templates:
        print("No templates found.")
        return
    print("Bundled templates:")
    for name in templates:
        print(f"  - {name}")


def resolve_template_paths(raw_paths: List[str]) -> List[str]:
    resolved = []
    for raw in raw_paths:
        candidate = resolve_path(raw, os.getcwd())
        if os.path.exists(candidate):
            resolved.append(candidate)
            continue
        bundled = os.path.join(DEFAULT_TEMPLATES_DIR, raw)
        if os.path.exists(bundled):
            resolved.append(bundled)
            continue
        raise SystemExit(f"Error: template not found: {raw}")
    return resolved


def main(argv: List[str]) -> int:
    args = parse_args(argv)

    if args.list_templates:
        list_templates()
        return 0

    data_path = (
        DEFAULT_DATA_PATH if args.data is None else resolve_path(args.data, os.getcwd())
    )
    template_paths = resolve_template_paths(args.template)
    settings_path = resolve_path(args.settings, os.getcwd()) if args.settings else None

    settings = load_settings(DEFAULT_SETTINGS_PATH, template_paths, settings_path)
    data = load_data(data_path)

    if args.seed is not None:
        random.seed(args.seed)

    counts = settings["counts"]
    constraints = settings["constraints"]
    validate_data(data, counts["users"], constraints["min_user_names"])

    security = settings["security"]
    auth = settings["auth"]

    flag_secret = os.getenv("FLAG_HMAC_SECRET", security["flag_hmac_secret_default"])
    bcrypt_cost = int(os.getenv("BCRYPT_COST", str(security["bcrypt_cost"])))
    output_file = os.getenv("OUTPUT_SQL_FILE", settings["output"]["file"])
    if args.output:
        output_file = args.output

    print("About to generate dummy SQL data.")
    print(f"Output file: {output_file}")
    print(f"Users: {counts['users']} (including admin)")
    print(f"Teams: {len(data['teams'])}")
    print(f"Challenges: {len(data['challenges'])}")
    print(f"Registration keys: {counts['registration_keys']}")
    proceed = input("Type 'Y' to continue: ").strip()
    if proceed != "Y":
        print("Aborted.")
        return 0

    teams = generate_teams(data["teams"], settings["timing"])
    team_ids = list(range(1, len(teams) + 1))
    users = generate_users(
        data["users"],
        counts["users"],
        team_ids,
        settings["timing"],
        settings["probabilities"],
        auth,
        bcrypt_cost,
    )
    challenges = generate_challenges(
        data["challenges"],
        settings["timing"],
        constraints,
        flag_secret,
    )
    registration_keys = generate_registration_keys(
        len(users),
        team_ids,
        settings["timing"],
        settings["probabilities"],
        counts["registration_keys"],
    )
    submissions = generate_submissions(
        users,
        data["challenges"],
        settings["timing"],
        settings["probabilities"],
        flag_secret,
    )

    write_sql_file(
        output_file,
        teams,
        users,
        challenges,
        registration_keys,
        submissions,
        {
            "flag_hmac_secret": flag_secret,
            "bcrypt_cost": bcrypt_cost,
            "default_password": auth["default_password"],
            "admin_email": auth["admin"]["email"],
            "admin_password": auth["admin"]["password"],
        },
    )

    print("\nSummary")
    print(f"- Output: {output_file}")
    print(f"- Teams: {len(teams)}")
    print(f"- Users: {len(users)}")
    print(f"- Challenges: {len(challenges)}")
    print(f"- Registration keys: {len(registration_keys)}")
    print(f"- Submissions: {len(submissions)}")
    print("\nLoad command")
    print(
        f"  PGPASSWORD=app_password psql -U app_user -d app_db -h localhost < {output_file}"
    )

    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
