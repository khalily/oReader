import time
import uuid

import grpc

from logging_config import bind_request_context, clear_request_context, get_logger

logger = get_logger(__name__)


class LoggingInterceptor:
    """gRPC server interceptor that logs RPC entry/exit with structured context.

    Mirrors the Go backend's RequestLoggerMiddleware:
    - Generates a request_id per RPC call
    - Logs method name, duration, and status
    - Binds request_id to structlog context so handler logs include it

    Handles all four gRPC RPC types:
    - unary-unary (e.g. ExtractMetadata)
    - unary-stream / server-streaming (e.g. Convert)
    - stream-unary / client-streaming
    - stream-stream
    """

    def intercept_service(self, continuation, handler_call_details):
        handler = continuation(handler_call_details)
        if handler is None:
            return None

        method_name = handler_call_details.method.split("/")[-1] if handler_call_details.method else "unknown"
        request_id = str(uuid.uuid4())[:8]

        if handler.request_streaming and handler.response_streaming:
            return self._wrap_stream_stream(handler, method_name, request_id)
        elif handler.request_streaming:
            return self._wrap_stream_unary(handler, method_name, request_id)
        elif handler.response_streaming:
            return self._wrap_unary_stream(handler, method_name, request_id)
        else:
            return self._wrap_unary_unary(handler, method_name, request_id)

    def _wrap_unary_unary(self, handler, method_name, request_id):
        original = handler.unary_unary

        def wrapped(request, context):
            bind_request_context(request_id=request_id, method=method_name)
            logger.info("RPC started", method=method_name, request_id=request_id)
            start = time.monotonic()
            try:
                result = original(request, context)
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.info("RPC completed", method=method_name, duration_ms=duration_ms)
                return result
            except Exception as e:
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.error("RPC failed", method=method_name, error=str(e), duration_ms=duration_ms)
                raise
            finally:
                clear_request_context()

        return grpc.unary_unary_rpc_method_handler(
            wrapped,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )

    def _wrap_unary_stream(self, handler, method_name, request_id):
        original = handler.unary_stream

        def wrapped(request, context):
            bind_request_context(request_id=request_id, method=method_name)
            logger.info("RPC started", method=method_name, request_id=request_id)
            start = time.monotonic()
            try:
                for response in original(request, context):
                    yield response
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.info("RPC completed", method=method_name, duration_ms=duration_ms)
            except Exception as e:
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.error("RPC failed", method=method_name, error=str(e), duration_ms=duration_ms)
                raise
            finally:
                clear_request_context()

        return grpc.unary_stream_rpc_method_handler(
            wrapped,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )

    def _wrap_stream_unary(self, handler, method_name, request_id):
        original = handler.stream_unary

        def wrapped(request_iterator, context):
            bind_request_context(request_id=request_id, method=method_name)
            logger.info("RPC started", method=method_name, request_id=request_id)
            start = time.monotonic()
            try:
                result = original(request_iterator, context)
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.info("RPC completed", method=method_name, duration_ms=duration_ms)
                return result
            except Exception as e:
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.error("RPC failed", method=method_name, error=str(e), duration_ms=duration_ms)
                raise
            finally:
                clear_request_context()

        return grpc.stream_unary_rpc_method_handler(
            wrapped,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )

    def _wrap_stream_stream(self, handler, method_name, request_id):
        original = handler.stream_stream

        def wrapped(request_iterator, context):
            bind_request_context(request_id=request_id, method=method_name)
            logger.info("RPC started", method=method_name, request_id=request_id)
            start = time.monotonic()
            try:
                for response in original(request_iterator, context):
                    yield response
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.info("RPC completed", method=method_name, duration_ms=duration_ms)
            except Exception as e:
                duration_ms = int((time.monotonic() - start) * 1000)
                logger.error("RPC failed", method=method_name, error=str(e), duration_ms=duration_ms)
                raise
            finally:
                clear_request_context()

        return grpc.stream_stream_rpc_method_handler(
            wrapped,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )
