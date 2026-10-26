import hashlib
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest

spec=importlib.util.spec_from_file_location('builder',Path(__file__).with_name('nervos-rootfs.py'))
builder=importlib.util.module_from_spec(spec);spec.loader.exec_module(builder)

class BuildRootfs(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        self.base=self.root/'base.tar.gz';self.daemon=self.root/'daemon';self.out=self.root/'out.tar.gz'
        elf=bytearray(32);elf[:6]=b'\x7fELF\x01\x01';elf[18]=40
        self.daemon.write_bytes(elf)
    def base_image(self,extra=None,omit=None):
        files={'usr/local/bin/techo5':b'old','etc/techo5-release':b'old version','etc/techo5/boot.sh':b'preserved recovery'}
        if omit:files.pop(omit)
        if extra:files.update(extra)
        with tarfile.open(self.base,'w:gz') as t:
            for name,data in files.items():
                info=tarfile.TarInfo('./'+name);info.size=len(data);info.uid=0;info.mode=0o755
                t.addfile(info,io.BytesIO(data))
        return hashlib.sha256(self.base.read_bytes()).hexdigest()
    def build(self,digest):
        return builder.assemble(self.base,self.daemon,self.out,'v1.6.9-nervos.1',digest)
    def test_only_daemon_and_label_replaced(self):
        self.build(self.base_image())
        with tarfile.open(self.out) as t:
            self.assertEqual(t.extractfile('./usr/local/bin/techo5').read(),self.daemon.read_bytes())
            self.assertEqual(t.extractfile('./etc/techo5/boot.sh').read(),b'preserved recovery')
            self.assertIn(b'v1.6.9-nervos.1',t.extractfile('./etc/techo5-release').read())
            self.assertEqual(t.getmember('./usr/local/bin/techo5').mode,0o755)
    def test_wrong_base_hash_refused(self):
        self.base_image()
        with self.assertRaises(ValueError):self.build('0'*64)
        self.assertFalse(self.out.exists())
    def test_missing_member_leaves_no_release(self):
        digest=self.base_image(omit='etc/techo5-release')
        with self.assertRaises(ValueError):self.build(digest)
        self.assertFalse(self.out.exists());self.assertFalse(Path(str(self.out)+'.partial').exists())
    def test_traversal_refused_without_extraction(self):
        digest=self.base_image(extra={'../escape':b'bad'})
        with self.assertRaises(ValueError):self.build(digest)
        self.assertFalse((self.root.parent/'escape').exists())
    def test_existing_output_preserved(self):
        digest=self.base_image();self.out.write_bytes(b'existing')
        with self.assertRaises(FileExistsError):self.build(digest)
        self.assertEqual(self.out.read_bytes(),b'existing')

if __name__=='__main__':unittest.main()
