import logging
import os
import sys

import structlog
from structlog.stdlib import ProcessorFormatter


def configure_logging(
    level: str | None = None,
    dev_mode: bool | None = None,
    stream: object | None = None,
) -> None:
    """Configure structured logging for the converter service.

    Production mode outputs JSON to stderr (matching Go backend's zerolog).
    Development mode outputs pretty console output.

    Args:
        level: Log level string (debug/info/warning/error). Defaults to LOG_LEVEL env var or "info".
        dev_mode: True for console output. Defaults to LOG_DEV env var or False.
        stream: Output stream for testing. Defaults to sys.stderr.
    """
    if level is None:
        level = os.getenv("LOG_LEVEL", "info")
    if dev_mode is None:
        dev_mode = os.getenv("LOG_DEV", "").lower() in ("true", "1", "yes")
    if stream is None:
        stream = sys.stderr

    # Map string level to logging constants
    level_map = {
        "debug": logging.DEBUG,
        "info": logging.INFO,
        "warning": logging.WARNING,
        "warn": logging.WARNING,
        "error": logging.ERROR,
        "fatal": logging.CRITICAL,
    }
    log_level = level_map.get(level.lower(), logging.INFO)
    logging.basicConfig(level=log_level, stream=stream, force=True)

    # Shared processors for both structlog and stdlib logging
    shared_processors = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
    ]

    if dev_mode:
        renderer = structlog.dev.ConsoleRenderer()
    else:
        renderer = structlog.processors.JSONRenderer()

    # Configure structlog
    structlog.configure(
        processors=[
            *shared_processors,
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ],
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=False,
    )

    # Configure root logger to use structlog's formatter
    # This ensures third-party libraries (grpc, magic_pdf) also output structured logs
    formatter = ProcessorFormatter(
        processors=shared_processors + [renderer],
    )
    handler = logging.StreamHandler(stream)
    handler.setFormatter(formatter)

    root_logger = logging.getLogger()
    root_logger.handlers.clear()
    root_logger.addHandler(handler)
    root_logger.setLevel(log_level)


def get_logger(name: str) -> structlog.stdlib.BoundLogger:
    """Get a structlog logger with the given module name.

    The name is propagated via stdlib's logging so that add_logger_name
    includes it in every log entry (matching Go backend's convention).
    """
    return structlog.get_logger(name)


def bind_request_context(**kwargs: object) -> None:
    """Bind key-value pairs to the current request context.

    Used by gRPC interceptor to set request_id, method, etc.
    These fields appear in all log statements within the request scope.
    """
    structlog.contextvars.bind_contextvars(**kwargs)


def clear_request_context() -> None:
    """Clear all request context variables."""
    structlog.contextvars.clear_contextvars()
