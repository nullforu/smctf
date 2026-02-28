
def escape_sql_string(value: str) -> str:
    return value.replace("'", "''")
