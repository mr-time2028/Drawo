# Production frontend image.
#
# Stage 1 builds the React/Vite app with Node.
# Stage 2 serves the compiled static files with Nginx and proxies API/WebSocket
# traffic to the Go backend. This means production servers need Docker only —
# no Node/npm installation on the host.

FROM node:22-alpine AS builder

WORKDIR /app

COPY frontend/app/package*.json ./
RUN npm ci

COPY frontend/app .

ARG VITE_API_BASE_URL=
ARG VITE_WS_URL=
ARG VITE_DEFAULT_LANGUAGE=fa
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_WS_URL=$VITE_WS_URL
ENV VITE_DEFAULT_LANGUAGE=$VITE_DEFAULT_LANGUAGE

RUN npm run build

FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
