#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import tempfile
import unittest
spec=importlib.util.spec_from_file_location('contracts', Path(__file__).with_name('check-generated-contracts.py'))
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class ContractCheck(unittest.TestCase):
    def test_compares_both_outputs_without_changing_client(self):
        with tempfile.TemporaryDirectory() as directory:
            expected=Path(directory)/'client'; (expected/'src').mkdir(parents=True)
            (expected/'openapi.json').write_text('{}')
            (expected/'src/schema.d.ts').write_text('export {};')
            (expected/'untracked.txt').write_text('keep')
            before={str(p.relative_to(expected)):p.read_bytes() for p in expected.rglob('*') if p.is_file()}
            def generator(destination):
                (destination/'src').mkdir()
                (destination/'openapi.json').write_text('{}')
                (destination/'src/schema.d.ts').write_text('export {};')
            self.assertEqual(module.verify(expected, generator), [])
            for filename in ['openapi.json','src/schema.d.ts']:
                original=(expected/filename).read_bytes()
                (expected/filename).write_text('intentional local edit')
                self.assertEqual(module.verify(expected, generator), [filename])
                self.assertEqual((expected/filename).read_text(), 'intentional local edit')
                (expected/filename).write_bytes(original)
            def failure(destination): raise RuntimeError('generation failed')
            with self.assertRaises(RuntimeError): module.verify(expected, failure)
            self.assertEqual(before,{str(p.relative_to(expected)):p.read_bytes() for p in expected.rglob('*') if p.is_file()})

if __name__=='__main__': unittest.main()
