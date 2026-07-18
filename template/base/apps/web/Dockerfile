# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e
FROM node:24.4.1-alpine3.22@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a AS build
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@11.0.7 --activate
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY apps/mobile/package.json apps/mobile/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/i18n/package.json packages/i18n/package.json
RUN pnpm install --frozen-lockfile
COPY apps/web apps/web
COPY packages/api-client packages/api-client
COPY packages/i18n packages/i18n
RUN pnpm --dir apps/web build
RUN pnpm --filter @__APP_SLUG__/web deploy --legacy --prod /prod

FROM node:24.4.1-alpine3.22@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a
ENV NODE_ENV=production HOST=0.0.0.0 PORT=3000
WORKDIR /app
COPY --from=build --chown=node:node /prod ./
USER node
EXPOSE 3000
CMD ["node", "build"]
