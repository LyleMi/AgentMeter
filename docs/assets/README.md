# Assets

Use synthetic or redacted data for public images.

- `screenshots/overview.png` is the README dashboard screenshot.
- `social-preview.png` is the GitHub/social link preview image.

Do not include real prompts, secrets, private paths, repository names, raw
session IDs, or unredacted audit evidence in public assets.

## Public Image Policy

- Keep public screenshots synthetic or redacted before publishing them.
- Keep the README screenshot in Git as one optimized current image. Replace
  `screenshots/overview.png` in place instead of adding dated screenshot
  history.
- Do not use Git LFS for these small public PNG assets. Prefer compression and
  keep routine screenshots roughly under 300 KB.
- GitHub Pages URLs are fine for latest/demo/social-preview assets, including
  `social-preview.png`, but do not use a Pages URL as the only README screenshot
  source when the image should stay versioned with the repository.
