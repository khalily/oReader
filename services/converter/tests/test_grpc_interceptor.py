import json
import os
import sys
import unittest
from io import StringIO
from unittest.mock import MagicMock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from interceptor import LoggingInterceptor  # noqa: E402
from logging_config import (  # noqa: E402
    clear_request_context,
    configure_logging,
    get_logger,
)


def _make_unary_unary_handler(fn):
    """Create a gRPC unary-unary handler."""
    import grpc

    return grpc.unary_unary_rpc_method_handler(fn)


def _make_unary_stream_handler(fn):
    """Create a gRPC server-streaming (unary-stream) handler."""
    import grpc

    return grpc.unary_stream_rpc_method_handler(fn)


def _make_stream_unary_handler(fn):
    """Create a gRPC client-streaming (stream-unary) handler."""
    import grpc

    return grpc.stream_unary_rpc_method_handler(fn)


def _make_stream_stream_handler(fn):
    """Create a gRPC bidirectional-streaming handler."""
    import grpc

    return grpc.stream_stream_rpc_method_handler(fn)


class _InterceptorTestBase(unittest.TestCase):
    """Base class for gRPC interceptor tests with isolated logging."""

    def setUp(self):
        self._stream = StringIO()
        configure_logging(level="debug", dev_mode=False, stream=self._stream)

    def tearDown(self):
        clear_request_context()

    def _get_log_entries(self):
        output = self._stream.getvalue().strip()
        if not output:
            return []
        return [json.loads(line) for line in output.split("\n") if line.strip()]


class TestGRPCInterceptorUnaryUnary(_InterceptorTestBase):
    """Test interceptor with unary-unary RPCs (e.g. ExtractMetadata)."""

    def test_interceptor_logs_rpc_entry_and_exit(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/ExtractMetadata"

        original_handler = _make_unary_unary_handler(lambda req, ctx: "response")
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)
        self.assertIsNotNone(result)

        result.unary_unary(None, None)

        entries = self._get_log_entries()
        events = [e.get("event") for e in entries]
        self.assertIn("RPC started", events)
        self.assertIn("RPC completed", events)

    def test_interceptor_generates_request_id(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/ExtractMetadata"

        original_handler = _make_unary_unary_handler(lambda req, ctx: "response")
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)
        result.unary_unary(None, None)

        entries = self._get_log_entries()
        for entry in entries:
            if entry.get("event") in ("RPC started", "RPC completed"):
                self.assertIn("request_id", entry)
                self.assertTrue(len(entry["request_id"]) > 0)

    def test_interceptor_extracts_method_name(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/ExtractMetadata"

        original_handler = _make_unary_unary_handler(lambda req, ctx: "response")
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)
        result.unary_unary(None, None)

        entries = self._get_log_entries()
        for entry in entries:
            if entry.get("event") in ("RPC started", "RPC completed"):
                self.assertEqual(entry.get("method"), "ExtractMetadata")

    def test_interceptor_logs_duration(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/ExtractMetadata"

        original_handler = _make_unary_unary_handler(lambda req, ctx: "response")
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)
        result.unary_unary(None, None)

        entries = self._get_log_entries()
        completed = [e for e in entries if e.get("event") == "RPC completed"]
        self.assertEqual(len(completed), 1)
        self.assertIn("duration_ms", completed[0])
        self.assertGreaterEqual(completed[0]["duration_ms"], 0)

    def test_interceptor_handler_sees_request_context(self):
        """Handler's log statements should include the request_id from interceptor."""
        captured = []

        def handler_logic(req, ctx):
            logger = get_logger("test_handler")
            logger.info("inside handler")
            captured.append("ok")
            return "response"

        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/ExtractMetadata"

        original_handler = _make_unary_unary_handler(handler_logic)
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)
        result.unary_unary(None, None)

        entries = self._get_log_entries()
        handler_log = [e for e in entries if e.get("event") == "inside handler"]
        self.assertEqual(len(handler_log), 1)
        self.assertIn("request_id", handler_log[0])
        self.assertEqual(captured, ["ok"])


class TestGRPCInterceptorUnaryStream(_InterceptorTestBase):
    """Test interceptor with server-streaming (unary-stream) RPCs (e.g. Convert)."""

    def test_streaming_handler_logs_completion(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/Convert"

        def stream_handler(req, ctx):
            yield "progress1"
            yield "progress2"

        original_handler = _make_unary_stream_handler(stream_handler)
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)

        list(result.unary_stream(None, None))

        entries = self._get_log_entries()
        events = [e.get("event") for e in entries]
        self.assertIn("RPC started", events)
        self.assertIn("RPC completed", events)

    def test_streaming_handler_error_logs_failure(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/paper.PaperConverter/Convert"

        def failing_stream_handler(req, ctx):
            yield "progress1"
            raise RuntimeError("conversion exploded")

        original_handler = _make_unary_stream_handler(failing_stream_handler)
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)

        with self.assertRaises(RuntimeError):
            list(result.unary_stream(None, None))

        entries = self._get_log_entries()
        events = [e.get("event") for e in entries]
        self.assertIn("RPC started", events)
        self.assertIn("RPC failed", events)

        failed = [e for e in entries if e.get("event") == "RPC failed"][0]
        self.assertIn("error", failed)
        self.assertEqual(failed["level"], "error")


class TestGRPCInterceptorStreamUnary(_InterceptorTestBase):
    """Test interceptor with client-streaming (stream-unary) RPCs."""

    def test_client_streaming_handler_logs_completion(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/test.Service/ClientStream"

        def handler(req_iter, ctx):
            return "aggregated"

        original_handler = _make_stream_unary_handler(handler)
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)

        response = result.stream_unary(iter([]), None)
        self.assertEqual(response, "aggregated")

        entries = self._get_log_entries()
        events = [e.get("event") for e in entries]
        self.assertIn("RPC started", events)
        self.assertIn("RPC completed", events)


class TestGRPCInterceptorStreamStream(_InterceptorTestBase):
    """Test interceptor with bidirectional-streaming RPCs."""

    def test_bidi_streaming_handler_logs_completion(self):
        interceptor = LoggingInterceptor()
        handler_call_details = MagicMock()
        handler_call_details.method = "/test.Service/BidiStream"

        def handler(req_iter, ctx):
            yield "response1"
            yield "response2"

        original_handler = _make_stream_stream_handler(handler)
        result = interceptor.intercept_service(lambda hcd: original_handler, handler_call_details)

        responses = list(result.stream_stream(iter([]), None))
        self.assertEqual(responses, ["response1", "response2"])

        entries = self._get_log_entries()
        events = [e.get("event") for e in entries]
        self.assertIn("RPC started", events)
        self.assertIn("RPC completed", events)


if __name__ == "__main__":
    unittest.main()
