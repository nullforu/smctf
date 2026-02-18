from pathlib import Path

exclude = {
    'internal/storage/s3.go',
    'internal/stack/client.go',
    'internal/http/handlers/types.go',
}
covered = 0
total = 0

if __name__ == '__main__':
    for line in Path('coverage.out').read_text().splitlines():
        if line.startswith('mode:') or not line.strip():
            continue

        parts = line.split()
        if len(parts) != 3:
            continue

        fname, stmts, count = parts
        fname = fname.split('"',1)[0]
        if fname in exclude:
            continue

        n = int(stmts); c = int(count)
        total += n
        if c > 0:
            covered += n

    pct = covered * 100.0 / total
    print(f'adjusted_statement_coverage={pct:.1f}% (covered={covered}, total={total})')
