import json
import os
import sys
import unittest
from io import StringIO

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from logging_config import (  # noqa: E402
    bind_request_context,
    clear_request_context,
    configure_logging,
    get_logger,
)


class TestConfigureLoggingJSON(unittest.TestCase):
    """Test production JSON logging mode."""

    def setUp(self):
        self._stream = StringIO()
        configure_logging(level="info", dev_mode=False, stream=self._stream)

    def _last_entry(self):
        output = self._stream.getvalue().strip()
        return json.loads(output.split("\n")[-1])

    def test_json_output_is_valid_json(self):
        logger = get_logger("test")
        logger.info("hello", key="value")
        data = self._last_entry()
        self.assertIn("timestamp", data)
        self.assertEqual(data["level"], "info")
        self.assertEqual(data["event"], "hello")
        self.assertEqual(data["key"], "value")

    def test_json_output_includes_logger_name(self):
        logger = get_logger("my_module")
        logger.info("test message")
        data = self._last_entry()
        self.assertEqual(data["logger"], "my_module")

    def test_json_error_log(self):
        logger = get_logger("test")
        logger.error("something broke", error_code=500)
        data = self._last_entry()
        self.assertEqual(data["level"], "error")
        self.assertEqual(data["error_code"], 500)


class TestConfigureLoggingDev(unittest.TestCase):
    """Test development console logging mode."""

    def test_dev_output_is_not_json(self):
        stream = StringIO()
        configure_logging(level="info", dev_mode=True, stream=stream)
        logger = get_logger("test")
        logger.info("hello dev")
        output = stream.getvalue()
        with self.assertRaises(json.JSONDecodeError):
            json.loads(output.strip())


class TestRequestContextBinding(unittest.TestCase):
    """Test request context binding via structlog contextvars."""

    def setUp(self):
        self._stream = StringIO()
        configure_logging(level="info", dev_mode=False, stream=self._stream)

    def tearDown(self):
        clear_request_context()

    def _last_entry(self):
        output = self._stream.getvalue().strip()
        return json.loads(output.split("\n")[-1])

    def test_bound_context_appears_in_log(self):
        bind_request_context(request_id="abc-123")
        logger = get_logger("test")
        logger.info("with context")
        data = self._last_entry()
        self.assertEqual(data["request_id"], "abc-123")

    def test_cleared_context_removed_from_log(self):
        bind_request_context(request_id="will-be-cleared")
        clear_request_context()
        logger = get_logger("test")
        logger.info("no context")
        data = self._last_entry()
        self.assertNotIn("request_id", data)

    def test_handler_sees_bound_context(self):
        """Simulates gRPC interceptor binding + handler logging."""
        bind_request_context(request_id="req-456", method="Convert")

        def handler():
            logger = get_logger("test_handler")
            logger.info("inside handler")

        handler()
        data = self._last_entry()
        self.assertEqual(data["request_id"], "req-456")
        self.assertEqual(data["method"], "Convert")


class TestLogLevelFiltering(unittest.TestCase):
    """Test log level filtering."""

    def test_warn_level_suppresses_info(self):
        stream = StringIO()
        configure_logging(level="warning", dev_mode=False, stream=stream)
        logger = get_logger("test")
        logger.info("should not appear")
        logger.warning("should appear")
        output = stream.getvalue().strip()
        lines = [line for line in output.split("\n") if line.strip()]
        self.assertEqual(len(lines), 1)
        data = json.loads(lines[0])
        self.assertEqual(data["level"], "warning")


if __name__ == "__main__":
    unittest.main()
