#!/bin/sh
amass intel -org $1 | awk '{print $1}' | cut -d, -f1
