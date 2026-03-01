#!/usr/bin/env python3

import argparse
import os
import sys
from typing import Dict, List, Optional

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
SCRIPTS_DIR = os.path.dirname(BASE_DIR)
if SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, SCRIPTS_DIR)

from data_loader import load_data, validate_data
from generator import (
    apply_challenge_pod_spec_paths,
    generate_challenges,
    generate_divisions,
    generate_teams,
    generate_users,
)
from sql_writer import write_sql_file
from sql_common.yaml_utils import deep_merge, resolve_path

DEFAULT_OUTPUT_FILE = "output.sql"
DEFAULT_SETTINGS = {
    "security": {
        "bcrypt_cost": 12,
    },
    "constraints": {
        "min_points_ratio": 0.2,
        "min_points_floor": 10,
    },
}


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate smctf SQL from YAML")
    parser.add_argument(
        "--data",
        required=True,
        help="Path to data YAML (teams/challenges).",
    )
    parser.add_argument(
        "--settings",
        help="Path to settings YAML (security/constraints) merged over defaults.",
    )
    parser.add_argument(
        "--output",
        help=f"Override output SQL file path (default: {DEFAULT_OUTPUT_FILE}).",
    )
    return parser.parse_args(argv)


def load_settings(path: Optional[str]) -> Dict[str, Dict[str, object]]:
    if not path:
        return DEFAULT_SETTINGS

    from sql_common.yaml_utils import load_yaml

    resolved = resolve_path(path, os.getcwd())
    user_settings = load_yaml(resolved)
    if not isinstance(user_settings, dict):
        raise SystemExit("Error: settings YAML root must be a mapping")
    return deep_merge(DEFAULT_SETTINGS, user_settings)


def main(argv: List[str]) -> int:
    args = parse_args(argv)

    data_path = resolve_path(args.data, os.getcwd())
    settings = load_settings(args.settings)
    data = load_data(data_path)
    if "divisions" not in data:
        raise SystemExit("Error: divisions must be provided in data YAML")
    validate_data(data)

    apply_challenge_pod_spec_paths(
        data.get("challenges", []), os.path.dirname(data_path)
    )

    security = settings["security"]
    constraints = settings["constraints"]

    bcrypt_cost = int(os.getenv("BCRYPT_COST", str(security["bcrypt_cost"])))
    output_file = args.output or DEFAULT_OUTPUT_FILE

    id_offset = 1
    divisions = generate_divisions(data["divisions"], id_offset=id_offset)
    division_map = {division["name"]: division["id"] for division in divisions}
    default_division = data["divisions"][0]
    teams = generate_teams(
        data["teams"],
        division_map,
        default_division,
        id_offset=id_offset,
    )
    users = generate_users(
        data["teams"],
        bcrypt_cost,
        id_offset=id_offset,
        team_id_offset=id_offset,
    )
    challenges = generate_challenges(data["challenges"], constraints, bcrypt_cost)

    write_sql_file(
        output_file,
        divisions,
        teams,
        users,
        challenges,
        {
            "bcrypt_cost": bcrypt_cost,
        },
    )

    print("SQL generated.")
    print(f"- Output: {output_file}")
    print(f"- Divisions: {len(divisions)}")
    print(f"- Teams: {len(teams)}")
    print(f"- Users: {len(users)}")
    print(f"- Challenges: {len(challenges)}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
