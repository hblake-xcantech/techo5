"""Exercise the real slotctl against disposable directory stores, never a device."""
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest

SCRIPT = Path(__file__).parent / 'linux' / 'slotctl'

class SlotRecovery(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store = Path(self.tmp.name) / 'store'
        (self.store / 'slots' / 'a').mkdir(parents=True)
        (self.store / '.techo5-store').touch()
        (self.store / 'active').write_text('a\n')
        (self.store / 'slots' / 'a.state').write_text('good\n')
        (self.store / 'slots' / 'a' / 'sentinel').write_text('known good')

    def run_slot(self, *args, ok=True):
        r = subprocess.run(['sh', str(SCRIPT), *args], env={**os.environ, 'STORE': str(self.store)}, capture_output=True, text=True)
        if ok:
            self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        else:
            self.assertNotEqual(r.returncode, 0)
        return r

    def archive(self, valid=True, board=None):
        p = Path(self.tmp.name) / 'rootfs.tar.gz'
        with tarfile.open(p, 'w:gz') as tar:
            paths = ['sbin/init','usr/local/bin/techo5','etc/techo5/boot.sh','etc/inittab'] if valid else ['broken']
            if board is not None:
                paths += ['etc/techo5/device.conf']
            for name in paths:
                data = ('T5_BOARD='+board+'\n').encode() if name.endswith('device.conf') else b'fixture\n'
                info = tarfile.TarInfo(name); info.size=len(data)
                tar.addfile(info, io.BytesIO(data))
        return p

    def assert_prior_preserved(self):
        self.assertEqual((self.store/'slots/a/sentinel').read_text(), 'known good')
        self.assertEqual((self.store/'slots/a.state').read_text().strip(), 'good')

    def test_install_preserves_prior_and_falls_back_after_three_trials(self):
        self.run_slot('install', str(self.archive()))
        self.assert_prior_preserved()
        self.assertEqual((self.store/'active').read_text().strip(),'b')
        for remaining in (2,1,0):
            self.assertEqual(self.run_slot('next').stdout.strip(),'b')
            self.assertEqual((self.store/'slots/b.state').read_text().strip(),f'trial {remaining}')
        self.assertEqual(self.run_slot('next').stdout.strip(),'a')
        self.assertEqual((self.store/'slots/b.state').read_text().strip(),'bad')
        self.assert_prior_preserved()

    def test_stage_does_not_switch_active(self):
        self.run_slot('install','--keep-active',str(self.archive()))
        self.assertEqual((self.store/'active').read_text().strip(),'a')
        self.assertEqual((self.store/'slots/b.state').read_text().strip(),'trial 3')
        self.assert_prior_preserved()

    def test_invalid_image_keeps_good_active_slot(self):
        self.run_slot('install',str(self.archive(valid=False)),ok=False)
        self.assertEqual((self.store/'active').read_text().strip(),'a')
        self.assert_prior_preserved()

    def test_board_mismatch_refused_before_inactive_slot_is_removed(self):
        (self.store/'.techo5-store').write_text('board=checkers\n')
        (self.store/'slots/b').mkdir()
        (self.store/'slots/b/sentinel').write_text('older backup')
        self.run_slot('install',str(self.archive(board='athena')),ok=False)
        self.assertEqual((self.store/'slots/b/sentinel').read_text(),'older backup')
        self.assert_prior_preserved()

    def test_no_good_slot_stays_in_rescue(self):
        (self.store/'slots/a.state').write_text('trial 0\n')
        self.run_slot('next',ok=False)
        self.assertEqual((self.store/'slots/a.state').read_text().strip(),'bad')

if __name__ == '__main__':
    unittest.main()
