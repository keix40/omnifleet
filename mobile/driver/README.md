# OmniFleet Driver (Expo)

React Native driver app scaffold for GPS ingestion.

## Status (stub)

- Expo project layout and env template are included for CI path filters.
- Production implementation will stream background GPS to `POST /api/v1/tracking/positions` using driver JWTs.

## Planned

- `@react-native-community/geolocation` background task
- Secure token storage via `expo-secure-store`
- Offline queue with retry

For the local vertical slice, use `scripts/gps-simulator` instead of the mobile app.
