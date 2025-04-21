# Use the latest Node.js LTS as the base image
FROM node:lts-alpine AS builder

# Set working directory
WORKDIR /app

ARG NEXT_PUBLIC_RPC_ADDRESS="__MY_NEXT_APP_RPC_PLACEHOLDER__"

# Copy package files
COPY package.json yarn.lock ./

# Install dependencies using Yarn
RUN yarn install 

# Copy the rest of the application
COPY . .

# Build the Next.js application
RUN yarn build

# Use a minimal Node.js runtime for production
FROM node:lts-alpine AS runner

# Set working directory
WORKDIR /app

# Install only production dependencies
COPY --from=builder /app/package.json /app/yarn.lock ./
RUN yarn install --production

# Copy the built Next.js application
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/next.config.js ./

COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose the port that your application will run on
EXPOSE 3000

# Command to start the application
ENTRYPOINT [ "/app/entrypoint.sh" ]
CMD ["yarn", "start"]

ARG GIT_URL=unspecified
ARG GIT_COMMIT=unspecified
ARG GIT_BRANCH=unspecified
ARG DATE=unspecified
ARG TAG=latest

LABEL org.opencontainers.image.vendor="SoC.one"
LABEL org.opencontainers.image.licenses="Proprietary"
LABEL org.opencontainers.image.description="SoC.one Payment Network"
LABEL org.opencontainers.image.documentation="https://soc.one"
LABEL org.opencontainers.image.source=$GIT_URL
LABEL org.opencontainers.image.revision=$GIT_COMMIT
LABEL org.opencontainers.image.branch=$GIT_BRANCH
LABEL org.opencontainers.image.date=$DATE
LABEL org.opencontainers.image.version=$TAG