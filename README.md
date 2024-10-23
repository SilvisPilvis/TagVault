<!-- # ![App icon: ](./icon.png) Tag Vault -->
<img src="./icon.png" alt="icon" width="84" height="84"/> <bold>A simple file management app with tags</bold>

<!-- ## A simple file management app with tags -->

[![wakatime](https://wakatime.com/badge/user/ee6ea1f8-6dfb-4436-aeb5-8339646633e5/project/4a604a7d-5a64-4fcd-a9bf-e2b84fe10b72.svg)](https://wakatime.com/badge/user/ee6ea1f8-6dfb-4436-aeb5-8339646633e5/project/4a604a7d-5a64-4fcd-a9bf-e2b84fe10b72)
![Static Badge](https://img.shields.io/badge/Krigga-approved-lightgreen?style=flat-square)

You can create tags, add them to images and search images by tags.
To make tags more stylish you can add colors to them.

To make this app possible I used Golang FyneGUI (not the best choice but it works for now) and SQLite3 for tag and options storage.

This apps current amazing features are:

- A Loading bar (much wow)
- Image loading/caching in the background
- In-App Fullscreen Image Viewing
- Automatic image discovery
- Ability to add multiple tags to single image
- Ability to blacklist files and folders
- Moved files persist tags
- Search by tag date or name
- Meta tags [PNG, JPG, Date Added]
- On first launch checks the Users picture directory to not freeze the program

Coming soon:

- [x] Multi select
- [x] Archive
- [x] Compress
- [x] Encrypt
- [x] GIFs will GIF (GIFs now GIF)
- [x] Convert
- [ ] Sorting (Currently sorts by date added)

Current supported image types:

- [x] JPG/JPEG
- [x] PNG
- [x] BMP
- [x] GIF
- [x] HEIC
- [x] TIFF
- [x] WEBP
- [x] AVIF
- [x] QOI
- [x] SVG   Only Viewing
- [ ] RAW   No
- [ ] JPEGXL   Someday... Maybe... Possibly...?

## App Demo Images

### Default app state

![App window demo: ](./demo.png)

### App with sidebar open

![App tag demo: ](./demo-tags.png)
