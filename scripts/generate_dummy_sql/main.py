#!/usr/bin/env python3

import argparse
import os
import random
import sys
from typing import List

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
SCRIPTS_DIR = os.path.dirname(BASE_DIR)
if SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, SCRIPTS_DIR)

from config_loader import load_settings, resolve_path
from data_loader import load_data, validate_data
from generator import (
    generate_divisions,
    generate_challenges,
    generate_registration_keys,
    generate_submissions,
    generate_teams,
    generate_users,
)
from sql_writer import write_sql_file

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


def load_text_file(path: str) -> str:
    with open(path, "r", encoding="utf-8") as f:
        return f.read().rstrip("\n")


def apply_challenge_vm_spec_paths(challenges: List[dict], base_dir: str) -> None:
    for chal in challenges:
        vm_spec_path = chal.get("vm_spec_path")
        if not vm_spec_path:
            continue
        resolved = resolve_path(vm_spec_path, base_dir)
        if not os.path.exists(resolved):
            raise SystemExit(
                f"Error: challenge sandbox spec file not found: {vm_spec_path}"
            )
        chal["vm_spec"] = load_text_file(resolved)


def normalize_divisions(raw_divisions: object, team_specs: List[dict]) -> List[str]:
    divisions: List[str] = []

    def add(name: str) -> None:
        if name not in divisions:
            divisions.append(name)

    if isinstance(raw_divisions, list):
        for entry in raw_divisions:
            if isinstance(entry, str) and entry.strip():
                add(entry.strip())

    for team in team_specs:
        division = team.get("division")
        if isinstance(division, str) and division.strip():
            add(division.strip())

    if "Admin" not in divisions:
        divisions.insert(0, "Admin")
    else:
        divisions = ["Admin"] + [name for name in divisions if name != "Admin"]

    return divisions


def normalize_teams(raw_teams: List[object], divisions: List[str]) -> List[dict]:
    teams: List[dict] = []
    non_admin_divisions = [name for name in divisions if name != "Admin"]
    fallback_division = non_admin_divisions[0] if non_admin_divisions else "Admin"
    round_robin_idx = 0

    for entry in raw_teams:
        if isinstance(entry, str):
            name = entry.strip()
            if not name:
                continue
            division = None
        else:
            name = str(entry.get("name", "")).strip()
            if not name:
                continue
            division = entry.get("division")
            division = str(division).strip() if isinstance(division, str) else None

        if name == "Admin":
            continue

        if not division:
            if non_admin_divisions:
                division = non_admin_divisions[round_robin_idx % len(non_admin_divisions)]
                round_robin_idx += 1
            else:
                division = fallback_division
        teams.append({"name": name, "division": division})

    return teams


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
    apply_challenge_vm_spec_paths(
        data.get("challenges", []), os.path.dirname(data_path)
    )

    if args.seed is not None:
        random.seed(args.seed)

    counts = settings["counts"]
    constraints = settings["constraints"]
    bootstrap = settings.get("bootstrap", {})
    use_bootstrap_admin = bool(bootstrap.get("use_bootstrap_admin", True))
    non_admin_count = max(0, counts["users"] - 1)
    validate_data(data, non_admin_count, constraints["min_user_names"])

    security = settings["security"]
    auth = settings["auth"]
    admin_team_name = "Admin"
    vm_config = settings.get("vm", {})
    files_config = settings.get("files", {})
    vm_spec = ""
    vm_spec_path = vm_config.get("vm_spec_path")
    if vm_config.get("enabled", False) and int(
        vm_config.get("random_challenge_count", 0)
    ) > 0 and not vm_spec_path:
        raise SystemExit(
            "Error: vm.vm_spec_path is required when vm is enabled"
        )
    if vm_spec_path:
        resolved_vm_spec_path = resolve_path(vm_spec_path, os.getcwd())
        if not os.path.exists(resolved_vm_spec_path):
            raise SystemExit(f"Error: sandbox spec file not found: {vm_spec_path}")
        vm_spec = load_text_file(resolved_vm_spec_path)

    bcrypt_cost = int(os.getenv("BCRYPT_COST", str(security["bcrypt_cost"])))
    output_file = os.getenv("OUTPUT_SQL_FILE", settings["output"]["file"])
    if args.output:
        output_file = args.output

    raw_teams = list(data["teams"])
    team_specs = []
    for entry in raw_teams:
        if isinstance(entry, str):
            team_specs.append({"name": entry})
        elif isinstance(entry, dict):
            team_specs.append(entry)
        else:
            raise SystemExit("Error: team entries must be strings or mappings")

    divisions = normalize_divisions(data.get("divisions"), team_specs)
    normalized_teams = normalize_teams(raw_teams, divisions)
    team_specs = [{"name": admin_team_name, "division": "Admin"}] + normalized_teams

    print("About to generate dummy SQL data.")
    print(f"Output file: {output_file}")
    admin_note = (
        "1 bootstrapped admin + "
        if use_bootstrap_admin
        else "including generated admin, "
    )
    print(f"Users: {counts['users']} ({admin_note}{non_admin_count} generated)")
    print(f"Divisions: {len(divisions)}")
    print(f"Teams: {len(team_specs)}")
    print(f"Challenges: {len(data['challenges'])}")
    print(f"Registration keys: {counts['registration_keys']}")
    proceed = input("Type 'Y' to continue: ").strip()
    if proceed != "Y":
        print("Aborted.")
        return 0

    division_rows = generate_divisions(divisions, settings["timing"])
    teams = generate_teams(team_specs, settings["timing"])
    team_ids = list(range(1, len(teams) + 1))
    admin_team_id = team_ids[0]
    non_admin_team_ids = team_ids[1:]
    users = generate_users(
        data["users"],
        counts["users"],
        admin_team_id,
        non_admin_team_ids,
        settings["timing"],
        settings["probabilities"],
        auth,
        bcrypt_cost,
        include_admin=True,
    )
    user_ids = list(range(1, len(users) + 1))
    challenges = generate_challenges(
        data["challenges"],
        settings["timing"],
        constraints,
        bcrypt_cost,
        vm_config,
        vm_spec,
        files_config,
    )
    registration_keys, registration_key_uses = generate_registration_keys(
        user_ids,
        non_admin_team_ids,
        settings["timing"],
        settings["probabilities"],
        counts["registration_keys"],
        created_by=1,
    )
    team_division_map = {
        team_id: team[1] for team_id, team in zip(team_ids, teams, strict=True)
    }
    for team_id, division in team_division_map.items():
        if not isinstance(division, str) or not division.strip():
            raise SystemExit(
                f"Error: team id {team_id} is missing a valid division assignment"
            )
    submissions = generate_submissions(
        users,
        data["challenges"],
        team_division_map,
        settings["timing"],
        settings["probabilities"],
        start_user_id=1,
        skip_first_user=True,
    )

    write_sql_file(
        output_file,
        division_rows,
        teams,
        users,
        challenges,
        registration_keys,
        registration_key_uses,
        submissions,
        {
            "bcrypt_cost": bcrypt_cost,
            "default_password": auth["default_password"],
            "admin_email": auth["admin"]["email"],
            "admin_password": auth["admin"]["password"],
            "include_admin": not use_bootstrap_admin,
            "bootstrap_mode": use_bootstrap_admin,
        },
    )

    print("\nSummary")
    print(f"- Output: {output_file}")
    print(f"- Divisions: {len(division_rows)}")
    print(f"- Teams: {len(teams)}")
    print(f"- Users: {len(users)}")
    if use_bootstrap_admin:
        print("- Admin user will be bootstrapped separately")
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
