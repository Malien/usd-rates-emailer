#!/bin/sh

echo "0 12 * * * cd /rates-emailer; ./publish-newsletter >> /var/log/publish-newsletter.log 2>&1" | crontab -
crond -f &
./rates-emailer &
touch /var/log/publish-newsletter.log
tail -f /var/log/publish-newsletter.log
