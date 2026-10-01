#!/usr/bin/env python3
import unittest
from ci_changes import ALL, Plan, classify, select_diff_range

class Routing(unittest.TestCase):
    def test_known_changes(self):
        self.assertEqual(classify(['docs/mobile.md']), Plan())
        self.assertEqual(classify(['apps/api/internal/app/example/service.go']), Plan(acceptance=True, api_image=True))
        self.assertEqual(classify(['apps/mobile/app/index.tsx']), Plan(native=True))
        self.assertEqual(classify(['packages/client-core/src/index.ts']), Plan(acceptance=True, web_image=True, native=True))
    def test_uncertain_changes_run_everything(self):
        for paths in [[], ['new-component/file'], ['/absolute'], ['../escape'], ['docs/readme.md', 'unknown']]:
            with self.subTest(paths=paths): self.assertEqual(classify(paths), ALL)
    def test_release_evidence_is_full(self):
        sha='a'*40; base='b'*40
        self.assertIsNone(select_diff_range('push', 'refs/heads/main', base, '', sha))
        self.assertIsNone(select_diff_range('workflow_dispatch', '', '', '', sha))
        self.assertEqual(select_diff_range('pull_request', '', '', base, sha), f'{base}...{sha}')
        self.assertIsNone(select_diff_range('pull_request', '', '', 'bad', sha))

if __name__ == '__main__': unittest.main()
