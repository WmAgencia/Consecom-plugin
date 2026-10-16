# Runtime versioned in GitHub; no base64 Function start-command payload.
FROM oven/bun:1
WORKDIR /app
COPY index.tsx /app/index.tsx
# Existing Railway volume /data contains the live SQLite and OAuth state.
# Avoid UID mismatches when switching from Railway Function image.
USER root
EXPOSE 8080
CMD ["bun", "run", "/app/index.tsx"]
