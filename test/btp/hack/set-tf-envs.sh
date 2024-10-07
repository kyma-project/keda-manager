#!/bin/bash

echo "$BTP_ENV" > /tmp/.env
export $(cat /tmp/.env | xargs)
rm /tmp/.env

