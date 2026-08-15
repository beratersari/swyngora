from swyngora_ai.config import Settings
from swyngora_ai.serve import is_service_authorized


def test_auth_open_when_token_empty():
    assert is_service_authorized({}, "") is True
    assert Settings(_env_file=None).service_token == ""


def test_auth_requires_bearer():
    token = "s3cret"
    assert is_service_authorized({}, token) is False
    assert is_service_authorized({"Authorization": "Bearer nope"}, token) is False
    assert is_service_authorized({"Authorization": "Bearer s3cret"}, token) is True
    assert is_service_authorized({"X-AI-Token": "s3cret"}, token) is True
