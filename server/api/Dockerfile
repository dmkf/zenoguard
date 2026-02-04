# Use official PHP 8.1 image with Apache
FROM php:8.1-apache

# Set working directory
WORKDIR /var/www/html

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    libpng-dev \
    libonig-dev \
    libxml2-dev \
    libzip-dev \
    unzip \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

# Install PHP extensions
RUN docker-php-ext-install \
    pdo \
    pdo_mysql \
    mbstring \
    exif \
    pcntl \
    bcmath \
    gd \
    zip

# Enable Apache mod_rewrite
RUN a2enmod rewrite

# Install Composer
COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

# Copy application files
COPY . /var/www/html/

# Set permissions
RUN chown -R www-data:www-data /var/www/html \
    && chmod -R 755 /var/www/html/storage \
    && chmod -R 755 /var/www/html/bootstrap/cache \
    && chmod -R 755 /var/www/html/storage/app \
    && chmod -R 755 /var/www/html/storage/framework

# Copy .env file if exists
COPY .env.example .env

# Generate application key if not exists
RUN php artisan key:generate --ansi || true

# Install dependencies
RUN composer install --no-dev --optimize-autoloader --no-interaction

# Cache configuration and routes
RUN php artisan config:cache \
    && php artisan route:cache \
    && php artisan view:cache \
    && php artisan event:cache

# Create supervisor configuration for Laravel Scheduler
RUN mkdir -p /etc/supervisor/conf.d
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Expose port 8888
EXPOSE 8888

# Start Apache and Supervisor in foreground
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
