#!/usr/bin/env python3
"""Verify generated API artifacts without rewriting the working tree."""
from pathlib import Path
import subprocess
import tempfile
import sys

ROOT = Path(__file__).resolve().parents[1]
FILES = ('openapi.json', 'src/schema.d.ts')

def generate(destination):
    (destination/'src').mkdir()
    result = subprocess.run(['go', 'run', './cmd/openapi'], cwd=ROOT/'apps/api', check=True, capture_output=True)
    (destination/'openapi.json').write_bytes(result.stdout)
    subprocess.run(['pnpm', '--dir', str(ROOT/'packages/api-client'), 'exec', 'openapi-typescript',
                    str(destination/'openapi.json'), '-o', str(destination/'src/schema.d.ts')], check=True)

def verify(expected, generator=generate):
    with tempfile.TemporaryDirectory(prefix='contract-check-') as directory:
        generated=Path(directory)
        generator(generated)
        return [name for name in FILES if not (expected/name).is_file()
                or (expected/name).read_bytes() != (generated/name).read_bytes()]

def main():
    try:
        drift=verify(ROOT/'packages/api-client')
        if drift:
            print('generated contract drift: '+', '.join(drift)+'; run make generate', file=sys.stderr)
            return 1
    except (OSError, subprocess.CalledProcessError) as error:
        print(f'contract generation failed: {error}', file=sys.stderr)
        return 1
    return 0

if __name__=='__main__': sys.exit(main())
