# Drawo Database Schema Documentation

This document describes the relational database schema for Drawo, implemented in PostgreSQL. All primary keys use UUIDs for enhanced security and distributed scalability.

## 1. Entity-Relationship Overview

The database is designed around a central **User** entity, with satellite tables handling specific domain responsibilities (Profiles, Settings, Statistics, etc.). Game-related data is persisted as historical summaries after ephemeral rooms close.

---

## 2. Table Definitions

### 2.1 `users`
The core account table.
- `id` (UUID, PK): Unique identifier.
- `username` (VARCHAR, Unique): Login name.
- `password_hash` (VARCHAR): Bcrypt hash of the password.
- `is_active` (BOOL): Account status.
- `is_superuser` (BOOL): Admin flag.

### 2.2 `profiles` (1:1 with `users`)
Extended user information.
- `user_id` (UUID, PK, FK): References `users.id`.
- `word_score`, `reputation_score`: Competitive and behavioral metrics.
- `avatar_url`, `email`, `phone`, `locale`, `theme`.

### 2.3 `friendships` & `friend_requests`
Social graph management.
- `friendships`: Composite PK `(user_id, friend_id)`. Junction table for mutual connections.
- `friend_requests`: Tracks invitation lifecycle (`pending`, `accepted`, `rejected`).

### 2.4 `game_histories`, `rounds`, `scores`
Persistent logs of completed games.
- `game_histories`: Summary of a finished session.
- `rounds`: Detail of each drawing turn (who drew what word).
- `scores`: Points earned by participants in a specific game.

### 2.5 `reports`
Moderation alerts.
- `reporter_id`, `reported_id`: Users involved.
- `reason`: Category (NSFW, Harassment, etc.).
- `details`: Reporter's description.

### 2.6 `achievements` & `player_statistics`
Long-term progression.
- `achievements`: Badges unlocked by users.
- `player_statistics`: Aggregated lifetime counters (Total Games, Wins, etc.).

### 2.7 `user_settings`
UI/UX preferences.
- `sound_enabled`, `music_enabled`, `theme`, `language_preference`.

### 2.8 `audit_logs`
Security and administrative audit trail.
- `action`, `details`, `ip`, `user_agent`.

---

## 3. Integrity Rules

- **Foreign Keys:** Enforced for all relationships.
- **On Delete Cascade:** Used for satellite tables (`profiles`, `settings`, `statistics`) so that deleting a user purges all associated data.
- **On Delete Set Null:** Used for historical logs (`game_histories`, `rounds`, `reports`) to preserve game history even if a participant deletes their account.
- **Constraints:** `no_self_friend` and `no_self_request` prevent users from interacting with themselves.

---

## 4. Discovery Index (Ephemeral Data)
**Note:** `Rooms` and `Players` (active session state) are **not** stored in this relational database. They are managed in Redis as ephemeral objects to support high-frequency updates and low-latency matchmaking.
