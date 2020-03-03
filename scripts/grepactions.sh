#!/bin/bash
grep -Eo '^[ ]*[<li>]*(/|url:|getJSON|postJSON|<a href=\"|window.location[ ]*=|action=|https\:\/\/.*site(.com|net)).*/[a-zA-Z0-9./?=_-]*\.php'
