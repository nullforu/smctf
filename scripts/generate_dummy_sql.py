#!/usr/bin/env python3

import os
import sys
import random
import hashlib
import hmac
from datetime import datetime, timedelta, timezone
from typing import List, Tuple

UTC = timezone.utc

FLAG_HMAC_SECRET = os.getenv('FLAG_HMAC_SECRET', 'change-me-too')
BCRYPT_COST = int(os.getenv('BCRYPT_COST', '12'))
OUTPUT_SQL_FILE = os.getenv('OUTPUT_SQL_FILE', 'dummy.sql')

try:
    import bcrypt
except ImportError:
    print("Error: bcrypt is required. Install it with: pip install bcrypt")
    sys.exit(1)


def hmac_flag(secret: str, flag: str) -> str:
    h = hmac.new(secret.encode(), flag.encode(), hashlib.sha256)
    return h.hexdigest()


def hash_password(password: str, cost: int = BCRYPT_COST) -> str:
    hashed = bcrypt.hashpw(password.encode(), bcrypt.gensalt(rounds=cost))
    return hashed.decode()


USER_NAMES = [
    ("김민준", "minjun.kim"),
    ("이서윤", "seoyoon.lee"),
    ("박지호", "jiho.park"),
    ("최수아", "sua.choi"),
    ("정예준", "yejun.jung"),
    ("강하윤", "hayoon.kang"),
    ("조도윤", "doyoon.cho"),
    ("윤서준", "seojun.yoon"),
    ("장시우", "siwoo.jang"),
    ("임지우", "jiwoo.lim"),
    ("한예은", "yeeun.han"),
    ("오현우", "hyunwoo.oh"),
    ("신지안", "jian.shin"),
    ("권도현", "dohyun.kwon"),
    ("황지윤", "jiyoon.hwang"),
    ("송민서", "minseo.song"),
    ("안준서", "junseo.ahn"),
    ("홍아린", "ahrin.hong"),
    ("김태양", "taeyang.kim"),
    ("김준영", "junyoung.kim"),
    ("박성민", "seongmin.park"),
    ("최윤호", "yunho.choi"),
    ("정소율", "soyul.jung"),
    ("강민재", "minjae.kang"),
    ("조유진", "yujin.cho"),
    ("윤채원", "chaewon.yoon"),
    ("장지훈", "jihoon.jang"),
    ("임수빈", "subin.lim"),
    ("한건우", "gunwoo.han"),
    ("오다은", "daeun.oh"),
    ("신우진", "woojin.shin"),
    ("권서아", "seoa.kwon"),
    ("황재현", "jaehyun.hwang"),
    ("송나은", "naeun.song"),
    ("안시현", "sihyun.ahn"),
    ("홍준혁", "junhyuk.hong"),
    ("김아윤", "ahyoon.kim"),
    ("이찬영", "chanyoung.lee"),
    ("박소현", "sohyun.park"),
    ("최지율", "jiyul.choi"),
    ("정태민", "taemin.jung"),
    ("강예린", "yerin.kang"),
    ("조승현", "seunghyun.cho"),
    ("윤아인", "ain.yoon"),
    ("장민혁", "minhyuk.jang"),
    ("임지원", "jiwon.lim"),
    ("한서영", "seoyoung.han"),
    ("오준영", "junyoung.oh"),
    ("신채은", "chaeeun.shin"),
    ("권동현", "donghyun.kwon"),
]

CHALLENGES = [
    ("Warmup", "Welcome to smctf! Can you find the flag in plain sight?", 50, "flag{w3lc0me_to_smctf_2024}", "Misc"),
    ("Easy Crypto", "Caesar cipher with a twist. Decode: Mjqqt1 Htsij", 100, "flag{caesar_cipher_easy}", "Crypto"),
    ("Web Basics", "Find the hidden admin page.", 100, "flag{hidden_in_robots_txt}", "Web"),
    ("Binary Exploitation 101", "Simple buffer overflow challenge.", 150, "flag{buffer_overflow_basics}", "Pwn"),
    ("Reverse Me", "Can you reverse this simple binary?", 150, "flag{rev3rse_3ng1neering}", "Reverse"),
    ("SQL Injection", "Classic SQLi. Admin login bypass.", 200, "flag{sqli_admin_bypass_2024}", "Web"),
    ("RSA Broken", "Small prime factorization problem.", 200, "flag{rsa_small_primes_weak}", "Crypto"),
    ("Format String", "Exploit the format string vulnerability.", 250, "flag{fmt_str1ng_vuln}", "Pwn"),
    ("XSS Challenge", "Steal the admin cookie.", 250, "flag{xss_cookie_stealer}", "Web"),
    ("Heap Overflow", "Advanced heap exploitation.", 300, "flag{heap_overflow_mastery}", "Pwn"),
    ("AES ECB Mode", "Exploit ECB mode weakness.", 300, "flag{ecb_mode_is_dangerous}", "Crypto"),
    ("Advanced Reversing", "Multi-stage obfuscated binary.", 350, "flag{advanced_reverse_2024}", "Reverse"),
    ("Race Condition", "Win the race to get the flag.", 350, "flag{race_condition_exploit}", "Web"),
    ("ROP Chain", "Return-oriented programming challenge.", 400, "flag{rop_chain_complete}", "Pwn"),
    ("Elliptic Curve", "Break weak elliptic curve cryptography.", 400, "flag{ecc_weak_curve_break}", "Crypto"),
    ("Kernel Exploitation", "Exploit a kernel vulnerability.", 450, "flag{kernel_pwn_master}", "Pwn"),
    ("Custom Protocol", "Reverse engineer a custom network protocol.", 450, "flag{custom_proto_reversed}", "Reverse"),
    ("JWT Forgery", "Forge a JWT token with algorithm confusion.", 450, "flag{jwt_alg_none_attack}", "Web"),
    ("Side Channel Attack", "Timing attack on crypto implementation.", 500, "flag{timing_attack_success}", "Crypto"),
    ("Final Boss", "Combine multiple vulnerabilities to get the flag.", 500, "flag{final_boss_defeated_2024}", "Misc"),
]


def generate_users(count: int) -> List[Tuple[str, str, str, str, str]]:
    users = []
    selected_names = random.sample(USER_NAMES, min(count, len(USER_NAMES)))
    
    base_time = datetime.now(UTC) - timedelta(hours=48)
    
    admin_password_hash = hash_password("admin123!")
    admin_time = base_time.strftime('%Y-%m-%d %H:%M:%S')
    users.append(("admin@smctf.com", "admin", admin_password_hash, "admin", admin_time))
    
    for i, (korean_name, username) in enumerate(selected_names):
        if i >= count - 1:  
            break
        
        email = f"{username}@example.com"
        password_hash = hash_password("password123")
        role = "user"
        created_at = (base_time + timedelta(hours=random.random() * 12))
        created_at_str = created_at.strftime('%Y-%m-%d %H:%M:%S')
        
        users.append((email, username, password_hash, role, created_at_str))
    
    return users


def generate_challenges() -> List[Tuple[str, str, int, str, bool, str]]:
    challenges = []
    base_time = datetime.now(UTC) - timedelta(hours=47)
    
    for i, (title, description, points, flag, category) in enumerate(CHALLENGES):
        flag_hash = hmac_flag(FLAG_HMAC_SECRET, flag)
        is_active = True
        created_at = (base_time + timedelta(minutes=i * 18))
        created_at_str = created_at.strftime('%Y-%m-%d %H:%M:%S')
        
        challenges.append((title, description, points, flag_hash, is_active, created_at_str))
    
    return challenges


def generate_submissions(user_count: int, challenge_count: int) -> List[Tuple[int, int, str, bool, str]]:
    submissions = []
    base_time = datetime.now(UTC) - timedelta(hours=42)
    
    
    for user_id in range(2, user_count + 1): 
        skill_level = random.betavariate(2, 5)
        attempt_count = random.randint(5, 15)
        attempted_challenges = set()

        for _ in range(attempt_count):

            challenge_weights = []
            for chal_id in range(1, challenge_count + 1):
                difficulty = chal_id / challenge_count
                weight = max(0.1, skill_level - difficulty + 0.3)
                challenge_weights.append(weight)
            
            chal_id = random.choices(range(1, challenge_count + 1), weights=challenge_weights)[0]
            attempted_challenges.add(chal_id)

        for chal_id in attempted_challenges:
            difficulty = chal_id / challenge_count

            submission_offset = timedelta(hours=random.random() * 42)
            submission_time = base_time + submission_offset
            
            solve_probability = max(0.1, skill_level - difficulty + 0.2)
            will_solve = random.random() < solve_probability
            
            if will_solve:
                wrong_attempts = random.choices([0, 1, 2], weights=[0.4, 0.4, 0.2])[0]
                for attempt in range(wrong_attempts):
                    wrong_time = submission_time - timedelta(minutes=random.randint(5, 60))
                    wrong_flag = f"flag{{wrong_attempt_{random.randint(1000, 9999)}}}"
                    wrong_hash = hmac_flag(FLAG_HMAC_SECRET, wrong_flag)
                    submissions.append((
                        user_id,
                        chal_id,
                        wrong_hash,
                        False,
                        wrong_time.strftime('%Y-%m-%d %H:%M:%S')
                    ))

                correct_flag = CHALLENGES[chal_id - 1][3] 
                correct_hash = hmac_flag(FLAG_HMAC_SECRET, correct_flag)
                submissions.append((
                    user_id,
                    chal_id,
                    correct_hash,
                    True,
                    submission_time.strftime('%Y-%m-%d %H:%M:%S')
                ))
            else:
                attempt_time = submission_time + timedelta(minutes=random.randint(0, 120))
                wrong_flag = f"flag{{incorrect_{random.randint(1000, 9999)}}}"
                wrong_hash = hmac_flag(FLAG_HMAC_SECRET, wrong_flag)
                submissions.append((
                    user_id,
                    chal_id,
                    wrong_hash,
                    False,
                    attempt_time.strftime('%Y-%m-%d %H:%M:%S')
                ))

    submissions.sort(key=lambda x: x[4])
    
    return submissions


def escape_sql_string(s: str) -> str:
    return s.replace("'", "''")


def generate_sql_file(output_file: str):
    print(f"Generating dummy data...")
    print(f"FLAG_HMAC_SECRET: {FLAG_HMAC_SECRET}")
    print(f"BCRYPT_COST: {BCRYPT_COST}")
    
    users = generate_users(50)
    challenges = generate_challenges()
    submissions = generate_submissions(len(users), len(challenges))
    
    print(f"Generated {len(users)} users")
    print(f"Generated {len(challenges)} challenges")
    print(f"Generated {len(submissions)} submissions")
    
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("-- smctf Dummy Data\n")
        f.write(f"-- Generated at: {datetime.now().isoformat()}\n")
        f.write(f"-- FLAG_HMAC_SECRET: {FLAG_HMAC_SECRET}\n")
        f.write(f"-- BCRYPT_COST: {BCRYPT_COST}\n")
        f.write("-- Default password for all users: password123\n")
        f.write("-- Admin credentials: admin@smctf.com / admin123!\n\n")
        
        f.write("-- Clear existing data\n")
        f.write("TRUNCATE TABLE submissions, challenges, users RESTART IDENTITY CASCADE;\n\n")
        
        f.write("-- Insert users\n")
        for email, username, password_hash, role, created_at in users:
            email_esc = escape_sql_string(email)
            username_esc = escape_sql_string(username)
            password_hash_esc = escape_sql_string(password_hash)
            role_esc = escape_sql_string(role)
            
            f.write(f"INSERT INTO users (email, username, password_hash, role, created_at, updated_at) VALUES ")
            f.write(f"('{email_esc}', '{username_esc}', '{password_hash_esc}', '{role_esc}', '{created_at}', '{created_at}');\n")
        
        f.write("\n")
        
        f.write("-- Insert challenges\n")
        for title, description, points, flag_hash, is_active, created_at in challenges:
            title_esc = escape_sql_string(title)
            description_esc = escape_sql_string(description)
            flag_hash_esc = escape_sql_string(flag_hash)
            
            f.write(f"INSERT INTO challenges (title, description, points, flag_hash, is_active, created_at) VALUES ")
            f.write(f"('{title_esc}', '{description_esc}', {points}, '{flag_hash_esc}', {is_active}, '{created_at}');\n")
        
        f.write("\n")
        
        f.write("-- Insert submissions\n")
        for user_id, challenge_id, provided, correct, submitted_at in submissions:
            provided_esc = escape_sql_string(provided)
            
            f.write(f"INSERT INTO submissions (user_id, challenge_id, provided, correct, submitted_at) VALUES ")
            f.write(f"({user_id}, {challenge_id}, '{provided_esc}', {correct}, '{submitted_at}');\n")
        
        f.write("\n")
        f.write("-- Update sequences\n")
        f.write("SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));\n")
        f.write("SELECT setval('challenges_id_seq', (SELECT MAX(id) FROM challenges));\n")
        f.write("SELECT setval('submissions_id_seq', (SELECT MAX(id) FROM submissions));\n")
    
    print(f"\nGenerated {output_file}")
    print(f"\nTo load the data:")
    print(f"  psql -U app_user -d app_db -h localhost < {output_file}")


if __name__ == "__main__":
    generate_sql_file(OUTPUT_SQL_FILE)
