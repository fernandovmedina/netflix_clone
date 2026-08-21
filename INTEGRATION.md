# ORCHESTRATION

## Multi-Agent Workflow

You are the **lead engineer and orchestrator** responsible for finishing the entire project end-to-end.

You have three Codex workers available.

### Codex 1

**Role:** Backend / implementation

Send tasks using:

`.claude/agents/1send.codex.sh "TASK"`

Read its output using:

`.claude/agents/1read.codex.sh`

### Codex 2

**Role:** Testing, QA, security review, and code review

Send tasks using:

`.claude/agents/2send.codex.sh "TASK"`

Read its output using:

`.claude/agents/2read.codex.sh`

### Codex 3

**Role:** Frontend / implementation / infrastructure

Send tasks using:

`.claude/agents/3send.codex.sh "TASK"`

Read its output using:

`.claude/agents/3read.codex.sh`

## Workflow

For every major feature:

1. Analyze the existing implementation before modifying it.
2. Understand the current architecture and business logic.
3. Create an implementation plan.
4. Split independent work between Codex 1 and Codex 3 when they can safely work in parallel.
5. Monitor their progress and read their outputs.
6. When implementation is complete, send the relevant changes to Codex 2 for testing and code review.
7. Codex 2 must check:

   * Functionality
   * Integration
   * Edge cases
   * Security
   * Performance
   * Concurrency issues
   * Horizontal-scaling compatibility
   * Responsive behavior where applicable
8. If Codex 2 finds problems, delegate fixes to Codex 1 or Codex 3.
9. Repeat implementation → review → fix until tests pass.
10. Perform your own final integration review.
11. Use Chrome via MCP to manually verify important user-facing workflows.
12. Report completion only after the complete system works locally.

Do not implement features yourself unless necessary.

Act primarily as the **lead engineer, architect, and orchestrator**.

---

# IMPORTANT (SEED VIDEO)
The seed video to show for movies and series is on: /seed/video/video.mp4

# MAIN TASK

Finish the entire project using the existing codebase and the seed data located in `/seed`.

The final result must run completely locally using Docker and PostgreSQL.

The architecture is intended for **horizontal scaling**, so preserve and improve that design. Services should remain stateless wherever practical and must not depend on local instance state for functionality that needs to survive load balancing.

Review the existing authentication load-balancing architecture and make sure authentication, sessions, API requests, media streaming, and other backend functionality work correctly when multiple backend instances are running.

Finish the backend and fully connect it to the frontend.

If frontend architecture or implementation needs to be upgraded to properly support the backend, make those changes.

---

# DATABASE

Migrate the project completely to **PostgreSQL**.

Remove Supabase dependencies and replace functionality previously provided by Supabase with the project's own backend implementation.

This includes, where applicable:

* Authentication
* User management
* Sessions
* Authorization
* Database access
* Storage references
* Any Supabase-specific API calls
* Any Supabase-specific frontend logic

Preserve the existing business logic.

Create appropriate:

* Tables
* Relationships
* Foreign keys
* Indexes
* Constraints
* Migrations

The system must work correctly with multiple backend instances.

---

# AUTHENTICATION

Rewrite the authentication system because Supabase authentication will no longer be used.

Implement the complete authentication workflow from scratch.

It should include:

* Registration
* Login
* Logout
* Secure password hashing
* Session/token handling
* Authorization
* Protected routes
* Authentication middleware
* Refresh/session expiration where appropriate
* Secure cookie handling where appropriate

Implement **Google OAuth** so users can sign in faster.

Everything required for Google OAuth should already exist in:

`/.env.local`

and/or:

`/microservices/oauth/auth`

Inspect the existing implementation before changing it.

Authentication must work correctly behind the project's load balancer and across multiple instances.

Do not design authentication in a way that requires a user to always reach the same backend instance.

---

# VIDEO PROCESSING AND STREAMING

The MP4 source videos are located in `/seed` inside their corresponding content folders.

Implement a production-style video processing and streaming pipeline similar in architecture to modern streaming platforms.

## Adaptive Bitrate Streaming

Do **not** simply serve the original MP4 files directly.

Use **adaptive bitrate streaming (ABR)**.

Prefer **HLS** unless the existing architecture provides a strong technical reason to use another standard.

Use FFmpeg or an equivalent production-grade tool to automatically transcode uploaded/source videos into multiple quality renditions.

Target these resolutions:

* 144p
* 240p
* 360p
* 480p
* 720p
* 1080p
* 1440p when the source quality supports it

Do **not upscale unnecessarily**.

For example:

* A 480p source should generate 144p, 240p, 360p, and 480p.
* A 1080p source should generate up to 1080p.
* A 1440p or higher source may generate the 1440p rendition.

Detect the source resolution automatically and generate only appropriate renditions.

Preserve the source aspect ratio.

Make sure generated dimensions are codec-compatible, including even-numbered dimensions where required.

## Encoding

Create reasonable bitrate profiles for each resolution instead of only resizing the video.

Configure appropriate:

* Video bitrate
* Maximum bitrate
* Buffer size
* Audio bitrate
* Codec
* GOP/keyframe interval
* Frame rate handling

Prefer broadly supported codecs such as:

* H.264 for video
* AAC for audio

Do not unnecessarily re-encode into exotic codecs that reduce browser/device compatibility.

Align keyframes between renditions so adaptive switching works correctly.

## HLS Segmentation

Videos must be delivered in **segments/chunks**, not as one giant MP4 download.

Generate:

* HLS master playlist
* Individual quality playlists
* Video segments
* Audio where appropriate

Use a sensible segment duration suitable for video-on-demand streaming, approximately **4–6 seconds**, unless testing demonstrates another value is better.

Example conceptual structure:

`movie-id/master.m3u8`

with renditions such as:

`144p/playlist.m3u8`

`240p/playlist.m3u8`

`360p/playlist.m3u8`

`480p/playlist.m3u8`

`720p/playlist.m3u8`

`1080p/playlist.m3u8`

`1440p/playlist.m3u8`

and their corresponding media segments.

The master playlist must advertise the available renditions correctly so the video player can automatically select quality according to the user's bandwidth and playback conditions.

## Playback

Update the frontend video player to support the implemented adaptive streaming format.

The player should:

* Start playback efficiently
* Automatically select an appropriate quality
* Adapt quality when bandwidth changes
* Avoid downloading the entire video
* Buffer only the required segments
* Allow seeking
* Recover gracefully from temporary segment/network failures
* Work on desktop and mobile browsers

Use native HLS support where appropriate and a compatible JavaScript HLS implementation where required.

Manual quality selection may also be implemented if it fits the existing UI, but automatic adaptive quality must work.

## Processing Pipeline

When an administrator uploads a movie or episode:

1. Validate the uploaded file.
2. Store/register the original source.
3. Create a video-processing job.
4. Process the video asynchronously.
5. Probe the source with FFprobe or equivalent.
6. Determine available output resolutions.
7. Transcode the required renditions.
8. Generate streaming segments and manifests.
9. Store metadata in PostgreSQL.
10. Mark the content as ready only after processing succeeds.

Do not make the HTTP upload request wait for the entire transcoding operation.

The database should track processing states such as:

* pending
* processing
* ready
* failed

Store useful metadata such as:

* Duration
* Source resolution
* Available qualities
* Processing status
* Streaming manifest location
* File size where useful
* Error information when processing fails

## Horizontal Scaling

Design the processing pipeline with horizontal scaling in mind.

Do not rely on an in-memory queue tied to a single backend instance.

Use an architecture where jobs can safely be processed by workers without duplicate processing when multiple instances exist.

Avoid storing required shared streaming state only on a backend container's ephemeral filesystem.

For the fully local Docker environment, use appropriate Docker volumes/shared storage.

Keep the storage abstraction clean enough that object storage such as S3-compatible storage could be introduced later without rewriting the entire application.

Streaming requests should be stateless and compatible with load balancing.

Implement correct HTTP behavior and caching headers for manifests and video segments.

Do not load complete video files into backend memory before sending them.

---

# ADMIN CONTENT MANAGEMENT

Create and finish the admin pages required to manage content.

Admins must be able to create/upload:

### Movies

Support:

* Metadata
* Images/posters where applicable
* Video upload
* Processing status
* Available streaming qualities
* Publishing state

### Series

Support:

* Series metadata
* Seasons
* Episodes
* Episode video uploads
* Processing status
* Available streaming qualities
* Publishing state

Uploaded videos must automatically enter the video processing pipeline.

The admin UI should clearly indicate:

`Uploading → Processing → Ready`

or:

`Failed`

Do not expose content to normal users as playable until its required processing has completed.

---

# PAYMENTS

Finish the payments section.

Currently there are multiple payment methods.

## Debit/Credit Card

Finish the existing card payment workflow.

Implement the discounts system in the backend and connect it to the frontend.

Discount validation and calculations must happen on the backend.

Never trust prices, totals, or discount calculations sent by the frontend.

## OXXO Payments

Create an algorithm to **simulate** the OXXO payment workflow locally.

Implement the discount system in the backend and connect it to this flow.

Clearly keep simulated payment behavior separated from real payment-provider integrations.

## Crypto Payments

Do not implement crypto payments.

Remove the crypto payment option from the frontend.

Remove dead crypto-specific code only when you have confirmed it is no longer required elsewhere.

---

# RESPONSIVE DESIGN

Make the complete website responsive for:

* Mobile
* Tablet
* Laptop
* Desktop

Check important pages manually through Chrome MCP at multiple viewport sizes.

Do not only rely on CSS assumptions.

Verify:

* Navigation
* Authentication
* Home/catalog
* Movie pages
* Series pages
* Video player
* Payments
* Admin dashboard
* Admin uploads
* Forms
* Modals

---

# DOCKER / LOCAL ENVIRONMENT

The entire project must work locally using Docker.

Finish and validate the Docker configuration for all required services.

This may include:

* Frontend
* Backend/API
* PostgreSQL
* Authentication/OAuth service
* Load balancer
* Video processing workers
* Shared media storage
* Any existing required microservices

The final environment should be reproducible.

Avoid manual setup steps when they can reasonably be automated.

Verify that the application still works when multiple backend instances are running behind the load balancer.

---

# SEED DATA

Use the existing data from:

`/seed`

Import the corresponding:

* Movies
* Series
* Episodes
* Metadata
* Images
* MP4 videos

Seeded MP4 files should go through or be compatible with the same video-processing pipeline used for newly uploaded content.

Do not create a completely separate playback architecture specifically for seed content.

---

# SECURITY

Codex 2 must specifically review:

* Password storage
* Authentication/session handling
* OAuth flow
* Authorization
* Admin authorization
* SQL injection
* File upload validation
* Path traversal
* Video/file access
* Payment calculations
* Discount abuse
* Secrets exposure
* CORS
* CSRF where applicable
* Cookie security
* Rate-limiting opportunities

Never expose secrets from `.env.local` to the frontend.

---

# WHAT TO DO

* Inspect and understand the existing architecture before rewriting components.
* Access Chrome via MCP to verify user-facing changes before committing them.
* Make Git commits for every important logical milestone.
* Use descriptive commit messages such as `feat:`, `fix:`, `refactor:`, `test:`, `chore:`, etc.
* Make the entire website responsive.
* Update `CLAUDE.md` to accurately document the new architecture and workflows.
* Replace Supabase authentication and database functionality.
* Connect everything to PostgreSQL.
* Implement Google OAuth.
* Finish the backend.
* Connect backend and frontend completely.
* Implement the adaptive video streaming architecture.
* Test with the actual MP4 files from `/seed`.
* Test multiple video source resolutions.
* Verify horizontal scaling.
* Verify everything locally through Docker.
* Run automated tests.
* Run integration tests.
* Perform manual browser verification.

---

# WHAT NOT TO DO

* Do not delete important code.
* Do not break existing business logic.
* Do not overwrite internal code without understanding its purpose.
* Do not write unnecessary code.
* Do not introduce unnecessary dependencies.
* Do not expose secrets.
* Do not hardcode credentials.
* Do not upscale low-resolution videos unnecessarily.
* Do not serve entire large video files through backend memory.
* Do not make transcoding block normal API requests.
* Do not depend on one specific backend instance for authentication or streaming.
* Do not use an in-memory-only processing queue if it breaks horizontal scaling.
* Do not commit generated video segments or large transcoded media files to Git.
* Do not implement crypto payments.
* Do not claim something works without testing it.

Even though Claude Code is running in auto mode, **ask for permission before performing database modifications that may modify, delete, reset, migrate, or destroy existing data.**

---

# DEFINITION OF DONE

Do not consider the project finished until:

1. The complete application starts locally through Docker.
2. PostgreSQL is fully integrated.
3. Supabase dependencies have been removed where they are no longer required.
4. Registration/login/logout work.
5. Google OAuth works.
6. Authentication works across multiple backend instances.
7. Seed data loads successfully.
8. Movies and series appear correctly.
9. Admins can create movies, series, seasons, and episodes.
10. Admins can upload videos.
11. Uploaded videos are processed asynchronously.
12. Videos are automatically transcoded into the appropriate available resolutions.
13. HLS manifests and segments are generated correctly.
14. Adaptive bitrate playback works.
15. Seeking works.
16. Playback works on desktop and mobile.
17. The player changes quality according to network conditions.
18. Card payment flow works locally as intended.
19. OXXO simulation works.
20. Discounts are validated by the backend.
21. Crypto payment UI has been removed.
22. The UI is responsive.
23. Automated tests pass.
24. Codex 2 completes the final review.
25. Chrome MCP verification passes.
26. Horizontal scaling has been tested.
27. `CLAUDE.md` reflects the final architecture.
28. Important milestones have appropriate Git commits.

---

# FEEDBACK

If something is genuinely ambiguous or requires a decision that could significantly change the architecture, ask through the Claude Code terminal.

Otherwise, inspect the existing project, make the safest reasonable engineering decision, document it, and continue without unnecessary interruptions.
