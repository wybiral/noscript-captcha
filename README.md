# noscript-captcha
A noscript image recognition captcha prototype.

This is a prototype meant to experiment with the concept of implementing a
ReCAPTCHA-style image recognition UI without using any JavaScript. Images aren't
included in this repo for copyright reasons but if you want to run this you'll
need to create a folder named /img filled with files c0-c999.jpg and d0-d999.jpg
that correspond with 1000 cat pictures and 1000 dog pictures.

Seriously don't use this anywhere other than testing until it gets a major
rewrite to clean up some of the potential concurrency issues and memory usage.