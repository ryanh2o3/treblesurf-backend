import unittest
from update_lambda_environment import merge

class MergeTest(unittest.TestCase):
    def test_preserves_unrelated_and_empty_values(self):
        existing = {"JWT_SECRET": "existing", "CUSTOM_SETTING": "keep", "OBSERVABILITY_ENABLED": "true"}
        actual = merge(existing, {"JWT_SECRET": "", "GO_ENV": "production", "OBSERVABILITY_ENABLED": "false"}, True)
        self.assertEqual(actual, {**existing, "GO_ENV": "production", "OBSERVABILITY_ENABLED": "false"})
        self.assertEqual(existing["OBSERVABILITY_ENABLED"], "true")

    def test_websocket_does_not_inherit_api_telemetry(self):
        self.assertNotIn("OBSERVABILITY_ENABLED", merge({}, {"OBSERVABILITY_ENABLED": "true"}))

if __name__ == "__main__":
    unittest.main()
