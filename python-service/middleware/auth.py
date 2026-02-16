"""Bearer token authentication middleware."""

from fastapi import HTTPException, Security
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from config import get_settings

# HTTPBearer extracts the token from the Authorization header
security = HTTPBearer()


def verify_token(
    credentials: HTTPAuthorizationCredentials = Security(security),
) -> str:
    """
    Dependency that validates the Bearer token.

    Args:
        credentials: The Authorization header credentials extracted by HTTPBearer.

    Returns:
        The validated token string.

    Raises:
        HTTPException: 401 if the token is invalid or missing.
    """
    settings = get_settings()

    if credentials.credentials != settings.api_token:
        raise HTTPException(
            status_code=401,
            detail="Invalid or expired token",
            headers={"WWW-Authenticate": "Bearer"},
        )

    return credentials.credentials
