#!/usr/bin/env python3
"""Build an undeployed NervOS rootfs from a signature-verified upstream base.

Only the daemon and release label change. No extraction, device access, install,
bootloader image, household config or key is part of this builder.
"""
import argparse
import copy
import gzip
import hashlib
import io
import os
from pathlib import Path, PurePosixPath
import re
import tarfile
from techo5lib import Release


def assemble(base, daemon, output, version, expected_base_hash):
    base, daemon, output = map(Path, (base, daemon, output))
    if not re.fullmatch(r'v\d+\.\d+\.\d+-nervos\.\d+', version):
        raise ValueError('Use an explicit vX.Y.Z-nervos.N release version')
    if hashlib.sha256(base.read_bytes()).hexdigest() != expected_base_hash:
        raise ValueError('Base differs from the signed upstream manifest')
    binary = daemon.read_bytes()
    if len(binary) < 20 or binary[:6] != b'\x7fELF\x01\x01' or int.from_bytes(binary[18:20], 'little') != 40:
        raise ValueError('Expected a little-endian 32-bit ARM ELF daemon')
    replacements = {
        'usr/local/bin/techo5': binary,
        'etc/techo5-release': f'NervOS Echo {version}; TECHO5-derived rootfs; upstream base v1.6.8\n'.encode(),
    }
    seen = set()
    partial = output.with_name(output.name + '.partial')
    if output.exists() or partial.exists():
        raise FileExistsError('Refusing to overwrite an existing release or partial build')
    output.parent.mkdir(parents=True, exist_ok=True)
    try:
        with tarfile.open(base, 'r:gz') as src, open(partial, 'xb') as f, gzip.GzipFile(filename='', mode='wb', fileobj=f, mtime=0) as gz, tarfile.open(fileobj=gz, mode='w|') as dst:
            for member in src:
                name = str(PurePosixPath(member.name))
                if name.startswith('/') or '..' in PurePosixPath(name).parts:
                    raise ValueError('Unsafe archive member path')
                if name in replacements:
                    if name in seen or not member.isfile():
                        raise ValueError('Ambiguous replacement member')
                    seen.add(name)
                    member = copy.copy(member)
                    payload = replacements[name]
                    member.size = len(payload)
                    member.mtime = 0
                    dst.addfile(member, io.BytesIO(payload))
                else:
                    dst.addfile(member, src.extractfile(member) if member.isfile() else None)
        if seen != set(replacements):
            raise ValueError('Base missing required daemon or release label')
        os.replace(partial, output)
    except BaseException:
        partial.unlink(missing_ok=True)
        raise
    return hashlib.sha256(output.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--daemon', required=True)
    p.add_argument('--output', required=True)
    p.add_argument('--version', required=True)
    p.add_argument('--cache', required=True)
    a = p.parse_args()
    release = Release('HuskerMinion/techo5', 'v1.6.8', a.cache)
    base = release.rootfs('arm')
    digest = assemble(base, a.daemon, a.output, a.version, release.manifest['rootfs']['arm']['sha256'])
    print(digest + '  ' + a.output)
    print('Built only: sign and verify the manifest before any separately coordinated slot install.')

if __name__ == '__main__':
    main()
