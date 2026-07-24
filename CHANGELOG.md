# Changelog

## [0.4.0](https://github.com/jonathanschwarzhaupt/home-blog/compare/v0.3.2...v0.4.0) (2026-07-24)


### Features

* add beaver icon and footer illustration ([#80](https://github.com/jonathanschwarzhaupt/home-blog/issues/80)) ([dc19ae9](https://github.com/jonathanschwarzhaupt/home-blog/commit/dc19ae981d254e2a1f2b59e7f26f8ad9eb2c79cf))

## [0.3.2](https://github.com/jonathanschwarzhaupt/home-blog/compare/v0.3.1...v0.3.2) (2026-07-23)


### Bug Fixes

* update home page text ([47878bb](https://github.com/jonathanschwarzhaupt/home-blog/commit/47878bb1cc87b83d137ff9aee398d6e8c66f80f2))

## [0.3.1](https://github.com/jonathanschwarzhaupt/home-blog/compare/v0.3.0...v0.3.1) (2026-07-22)


### Features

* Update home.templ text ([541a25a](https://github.com/jonathanschwarzhaupt/home-blog/commit/541a25ab05827dd0d6e3c79fc424d287cf223756))

## [0.3.0](https://github.com/jonathanschwarzhaupt/home-blog/compare/v0.2.0...v0.3.0) (2026-07-21)


### Features

* Add bulk Manage Order admin page for projects ([#70](https://github.com/jonathanschwarzhaupt/home-blog/issues/70)) ([06a006e](https://github.com/jonathanschwarzhaupt/home-blog/commit/06a006e0d3a057c6b08f9c4c3362944664f616ad))
* Add edit and delete to Projects ([#64](https://github.com/jonathanschwarzhaupt/home-blog/issues/64)) ([826153b](https://github.com/jonathanschwarzhaupt/home-blog/commit/826153b47240cbf9c41c8a267eeeb40e88af6a9c))
* Make About page editable with revision history; extract Skills into their own table ([#72](https://github.com/jonathanschwarzhaupt/home-blog/issues/72)) ([db9ef38](https://github.com/jonathanschwarzhaupt/home-blog/commit/db9ef3846188e0cdf513f82c41fa13b413ebb70b))


### Bug Fixes

* internal/vcs.Version() to report git tags instead of raw commit SHA ([#65](https://github.com/jonathanschwarzhaupt/home-blog/issues/65)) ([96e61be](https://github.com/jonathanschwarzhaupt/home-blog/commit/96e61bee752ed254177231826c8892ff0d2adb43))

## [0.2.0](https://github.com/jonathanschwarzhaupt/home-blog/compare/v0.1.0...v0.2.0) (2026-07-12)


### Features

* /admin landing page replaces scattered nav links (issue [#29](https://github.com/jonathanschwarzhaupt/home-blog/issues/29)) ([6d097b8](https://github.com/jonathanschwarzhaupt/home-blog/commit/6d097b85460a5f5c4026cf79aa313db2613e00aa))
* About Me page on blog (issue [#8](https://github.com/jonathanschwarzhaupt/home-blog/issues/8)) ([cf0c6e6](https://github.com/jonathanschwarzhaupt/home-blog/commit/cf0c6e685f92a9fbb01772d4bdf2bc10670cc794))
* add /posts index page (issue [#24](https://github.com/jonathanschwarzhaupt/home-blog/issues/24)) ([7fe2323](https://github.com/jonathanschwarzhaupt/home-blog/commit/7fe2323e5ee050b03739c79db438c50f39509ada))
* add a small back-link to admin pages and post/project detail pages ([4ef9c88](https://github.com/jonathanschwarzhaupt/home-blog/commit/4ef9c88a102ca7e7c77e853ab60b00f4b12e3c3e))
* add per-post admin kebab menu with Edit action (issue [#26](https://github.com/jonathanschwarzhaupt/home-blog/issues/26)) ([d83a633](https://github.com/jonathanschwarzhaupt/home-blog/commit/d83a6334b150e907963e301c9ead0ed9c561602d))
* add projects.created_at column + index (issue [#32](https://github.com/jonathanschwarzhaupt/home-blog/issues/32)) ([89970dd](https://github.com/jonathanschwarzhaupt/home-blog/commit/89970dde8fa4ba9098f8ce338184036a7f4c5463))
* add Prometheus metrics endpoint on a separate port (issue [#21](https://github.com/jonathanschwarzhaupt/home-blog/issues/21)) ([cd683ad](https://github.com/jonathanschwarzhaupt/home-blog/commit/cd683ad206d7d36cd56895cba7f5dfd2ac19612b))
* admin stats page reading the metrics registry directly (issue [#28](https://github.com/jonathanschwarzhaupt/home-blog/issues/28)) ([90e389d](https://github.com/jonathanschwarzhaupt/home-blog/commit/90e389d775b9378dee0679312cc040e74015b743))
* build and push images on tagged release ([#51](https://github.com/jonathanschwarzhaupt/home-blog/issues/51)) ([098c29a](https://github.com/jonathanschwarzhaupt/home-blog/commit/098c29ad0d47499f04d9dd1be1b5c4d3e754fb28))
* configure release-please ([#46](https://github.com/jonathanschwarzhaupt/home-blog/issues/46)) ([ed611a3](https://github.com/jonathanschwarzhaupt/home-blog/commit/ed611a3702283a1e083dd678f991b71971bdd7b5))
* delete posts from the admin kebab menu ([#42](https://github.com/jonathanschwarzhaupt/home-blog/issues/42)) ([317fd0e](https://github.com/jonathanschwarzhaupt/home-blog/commit/317fd0eab7c69f9ba28e1e2d9cf3e2f629ab8243))
* Dockerfile for cmd/blog ([#50](https://github.com/jonathanschwarzhaupt/home-blog/issues/50)) ([c697261](https://github.com/jonathanschwarzhaupt/home-blog/commit/c6972612d0cb7fc6b41bc049ed2c536a1f7a24d2))
* Dockerfile for cmd/migrate ([#47](https://github.com/jonathanschwarzhaupt/home-blog/issues/47)) ([bce5615](https://github.com/jonathanschwarzhaupt/home-blog/commit/bce5615996d9fff16b7d0c9ff2a5079cf15ed69e))
* featured content curation - schema, queries, admin page (issue [#23](https://github.com/jonathanschwarzhaupt/home-blog/issues/23)) ([5ab74d0](https://github.com/jonathanschwarzhaupt/home-blog/commit/5ab74d0610ea02301c603c8549834d2888af3e63))
* footer contact links (GitHub/LinkedIn/email) (issue [#40](https://github.com/jonathanschwarzhaupt/home-blog/issues/40)) ([c51fce0](https://github.com/jonathanschwarzhaupt/home-blog/commit/c51fce05a78f2558fb041fe88fa5d0f3e73c354e))
* footer quip becomes a hover-revealed Easter egg (issue [#41](https://github.com/jonathanschwarzhaupt/home-blog/issues/41)) ([53a24ea](https://github.com/jonathanschwarzhaupt/home-blog/commit/53a24eae94e288c8bea9c94a021f26889d3427bd))
* footer Stay Connected mailing-list CTA (issue [#38](https://github.com/jonathanschwarzhaupt/home-blog/issues/38)) ([9daf096](https://github.com/jonathanschwarzhaupt/home-blog/commit/9daf096728df8db5f2ab2383b832b3ca1dcdde27))
* free editing with optimistic concurrency (issue [#4](https://github.com/jonathanschwarzhaupt/home-blog/issues/4)) ([898eb4e](https://github.com/jonathanschwarzhaupt/home-blog/commit/898eb4e0688efda02758317b8a9d5f46bdaaa3be))
* implement compose & view a post (issue [#3](https://github.com/jonathanschwarzhaupt/home-blog/issues/3)) ([e5fb478](https://github.com/jonathanschwarzhaupt/home-blog/commit/e5fb478ea3a54992eed1d8e59519c8d9079d61a7))
* initial commit ([1ce2503](https://github.com/jonathanschwarzhaupt/home-blog/commit/1ce250311fff7e52fe426f5707a9ed244a26bcc6))
* log response status/duration as one line; add request correlation ID (issue [#20](https://github.com/jonathanschwarzhaupt/home-blog/issues/20)) ([eea1a95](https://github.com/jonathanschwarzhaupt/home-blog/commit/eea1a95a906de54d180604d92222388478e3e52c))
* merge cmd/blog-admin into cmd/blog behind a -features flag (issue [#18](https://github.com/jonathanschwarzhaupt/home-blog/issues/18)) ([73e74be](https://github.com/jonathanschwarzhaupt/home-blog/commit/73e74bec3bfa6e5dde530ec70644ec7a8adead73))
* per-client rate limiting on blog (issue [#5](https://github.com/jonathanschwarzhaupt/home-blog/issues/5)) ([36a408a](https://github.com/jonathanschwarzhaupt/home-blog/commit/36a408a689643248127beac45cde36dfcb77258d))
* personality theme — warm paper palette + boxy monospace detail ([703c653](https://github.com/jonathanschwarzhaupt/home-blog/commit/703c653376ffa4cdb91d00f33b7856c0928c6ae8))
* posts index pagination + newest/oldest sort + date-range filter (issue [#35](https://github.com/jonathanschwarzhaupt/home-blog/issues/35)) ([d0ed5a3](https://github.com/jonathanschwarzhaupt/home-blog/commit/d0ed5a3b7727a9c46f7544044e8043b58d38bee0))
* posts single-tag filtering + browse-all-tags (issue [#36](https://github.com/jonathanschwarzhaupt/home-blog/issues/36)) ([c60fa9c](https://github.com/jonathanschwarzhaupt/home-blog/commit/c60fa9c48a5f4849dd82c06e623213cb1e1ae4e4))
* progress ([8e46b79](https://github.com/jonathanschwarzhaupt/home-blog/commit/8e46b790d48cbf34692c2ec6bbcff34a828a6d21))
* project planning ([4ed19fc](https://github.com/jonathanschwarzhaupt/home-blog/commit/4ed19fc199646e11f7b56c722b855ea73cd99076))
* projects index pagination + newest/oldest sort + date-range filter (issue [#37](https://github.com/jonathanschwarzhaupt/home-blog/issues/37)) ([6af19c6](https://github.com/jonathanschwarzhaupt/home-blog/commit/6af19c6a7469555830666f00913ae65d8af0f0f7))
* Projects, many-to-many with Posts (issue [#7](https://github.com/jonathanschwarzhaupt/home-blog/issues/7)) ([badd892](https://github.com/jonathanschwarzhaupt/home-blog/commit/badd892680208b5b18172e29bb705789b7169f36))
* redesign Home as a landing/teaser page (issue [#25](https://github.com/jonathanschwarzhaupt/home-blog/issues/25)) ([95489a1](https://github.com/jonathanschwarzhaupt/home-blog/commit/95489a135839a8f68a12478acab1ce98c2818684))
* render post body as markdown at display time (issue [#19](https://github.com/jonathanschwarzhaupt/home-blog/issues/19)) ([e9ca001](https://github.com/jonathanschwarzhaupt/home-blog/commit/e9ca001b6c5bc33da758a6aec65fe8fd730737b0))
* replace About page placeholder with real profile photo ([dec9b2e](https://github.com/jonathanschwarzhaupt/home-blog/commit/dec9b2edb2259036b1fb6166687f6321d3f1d890))
* report status/mode/version from the healthcheck endpoint ([728ac9b](https://github.com/jonathanschwarzhaupt/home-blog/commit/728ac9b8ef70833c356238da903294babeb747e2))
* restyle About page with templui (issue [#13](https://github.com/jonathanschwarzhaupt/home-blog/issues/13)) ([76d76f0](https://github.com/jonathanschwarzhaupt/home-blog/commit/76d76f084e9cc5aa7f0f4bed799c924202f4b708))
* restyle blog-admin compose/edit forms with templui (issue [#14](https://github.com/jonathanschwarzhaupt/home-blog/issues/14)) ([38cc64b](https://github.com/jonathanschwarzhaupt/home-blog/commit/38cc64b8749c9ebb9d0a0f1dfbc3dbb4d7f7fc62))
* restyle blog-admin project-create form with templui (issue [#15](https://github.com/jonathanschwarzhaupt/home-blog/issues/15)) ([f256e29](https://github.com/jonathanschwarzhaupt/home-blog/commit/f256e29fef1b4e2d927b65b013a620de5a9c6a79))
* restyle home page + Post view with templui (issue [#11](https://github.com/jonathanschwarzhaupt/home-blog/issues/11)) ([a9deec2](https://github.com/jonathanschwarzhaupt/home-blog/commit/a9deec25934839bbe4691e91fad14b6402902045))
* restyle Projects index + Project view with templui (issue [#12](https://github.com/jonathanschwarzhaupt/home-blog/issues/12)) ([876063d](https://github.com/jonathanschwarzhaupt/home-blog/commit/876063d2005bc088e81c029ce452a6e0d5823d5b))
* RSS feed on blog (issue [#6](https://github.com/jonathanschwarzhaupt/home-blog/issues/6)) ([553438a](https://github.com/jonathanschwarzhaupt/home-blog/commit/553438ac096ee7a881bb03908ec42bebc08c28bc))
* settable published_at on post create/edit for backdating (issue [#33](https://github.com/jonathanschwarzhaupt/home-blog/issues/33)) ([66559d1](https://github.com/jonathanschwarzhaupt/home-blog/commit/66559d18b3150bfff811fada0fb91255b2c3114f))
* show published date next to tags ([#43](https://github.com/jonathanschwarzhaupt/home-blog/issues/43)) ([dfc5ffe](https://github.com/jonathanschwarzhaupt/home-blog/commit/dfc5ffe52557ca59e2231e2cd738bbac37dc145c))
* site-wide footer with rotating self-aware quips (issue [#27](https://github.com/jonathanschwarzhaupt/home-blog/issues/27)) ([0110fc5](https://github.com/jonathanschwarzhaupt/home-blog/commit/0110fc501c1e96b67827f70313a53bac9cb3faa1))
* styled 404 page, RSS discovery tag, favicon (quick UI wins) ([283f6b0](https://github.com/jonathanschwarzhaupt/home-blog/commit/283f6b00bbf4bc2f4cdf791b33c32f6413d3bd88))
* templui + Tailwind CSS pipeline, restyle shared layout (issue [#10](https://github.com/jonathanschwarzhaupt/home-blog/issues/10)) ([8bef7ff](https://github.com/jonathanschwarzhaupt/home-blog/commit/8bef7ffb855018bc83c6d110494f7d2a5d8e268f))
* write the real About page copy, tease it on Home ([cfcca03](https://github.com/jonathanschwarzhaupt/home-blog/commit/cfcca037c3cc5e24ba7b44e20f029b7e1cc459d2))


### Bug Fixes

* address code-review findings on walking skeleton ([2c6fab6](https://github.com/jonathanschwarzhaupt/home-blog/commit/2c6fab64689335a1f56754b4c696dc71ea9689c4))
* complete the module rename ([#45](https://github.com/jonathanschwarzhaupt/home-blog/issues/45)) ([89fbb63](https://github.com/jonathanschwarzhaupt/home-blog/commit/89fbb638a0652a604ccde2d26c53390782cc6303))
* expose MaxConnLifetime explicitly, matching Let's Go Further's pool config ([bd2313a](https://github.com/jonathanschwarzhaupt/home-blog/commit/bd2313ad9313b5ab3072041c2ef631b26188b8a6))
* stop logging static asset requests ([0ceeb38](https://github.com/jonathanschwarzhaupt/home-blog/commit/0ceeb3898d364ba1292353c4c30a2230b3504713))
