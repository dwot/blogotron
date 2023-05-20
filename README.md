# Blog-o-Tron

Blog-o-Tron (BOT) is an experimental interface between wordpress and openAI.  It allows for brainstorming ideas and authoring posts for a wordpress blog using OpenAI GPT-3.  It can connect to Dall-E or a Stable Diffusion instance to generate images for the post.  
It is a work in progress and is not ready for production use. 

## Configuration
### Settings 
#### Settings have been migrated to the database in the latest version along w/ prompts from the config file.
- WP_URL - The URL of the wordpress instance
- WP_USERNAME - The username of the wordpress user
- WP_PASSWORD - The application password of the wordpress user.  See https://www.paidmembershipspro.com/create-application-password-wordpress/
- BLOGOTRON_PORT - The port for the BOT web application.  Default is 8666
- BLOGOTRON_DB - The name of the database file.  Default is blogotron.db
- OPENAI_API_KEY - The API key for OpenAI.  See https://platform.openai.com/signup
- ENABLE_GPT4 - Enable GPT-4 API.  Default is false.  Must be granted access by OpenAI.
- UNSPLASH_ACCESS_KEY - The access key for Unsplash.  See https://unsplash.com/developers
- UNSPLASH_SECRET_KEY - The secret key for Unsplash.  See https://unsplash.com/developers
- IMG_MODE - The image generation mode.  Default is none.  Options are none, sd, or openai
- SD_URL - The URL for the Stable Diffusion instance.
- AUTO_POST_ENABLE - Enable auto posting.  Default is false.
- AUTO_POST_INTERVAL - The interval for auto posting in minutes.  Default is 24h.
- AUTO_POST_IMG_ENGINE - The image generation engine to use for auto posting.  Default is none.  Options are none, generate, or unsplash
- AUTO_POST_LEN - The length of the auto post.  Default is 500.
- AUTO_POST_STATE - The state of the auto post.  Default is draft.  Options are publish or draft.
- LOW_IDEA_THRESHOLD - The threshold for invoking idea generation.  Default is 0 which disables automatic idea generation.

### Build the Docker Image
1. git clone https://github.com/dwot/BlogoTron.git
2. cd BlogoTron
3. docker build -t blogotron:latest .

### Run the Docker Image
1. Create a dir to hold the database
2. docker run -d -p 8666:8666 -v /path/to/local/.env:/app/.env -v /path/to/local/config.yml:/app/config.yml -v /path/to/local/blogtron.db:/app/blogtron.db blogotron:latest
3. Browse to http://localhost:8666

### Run the Docker Compose
The docker compose will create a database, wordpress instance, redis instance and the blogotron instance.
1. Create the Docker Image w/ the above steps
2. Edit the docker-compose.yml and enter proper ports and paths for your environment
3. docker-compose up -d
4. Browse to the wordpress port and complete the install
5. Browse to the blogotron port and you should be set

## Usage
### Write
- From the Write screen you can author a blog post from a concept. You can use a vague concept and have the BOT create a title or provide an exact title and check "Use Concept as Title". 
Article Length and Post State (draft or publish) can be selected.
- If "Generate Image" is selected, a prompt can be entered and the enabled image generation engine (Dall-E via OpenAI API or Stable Diffusion) will be used to generate an image.
- The image will be saved to the media library and attached to the post. If a prompt is not entered and "Generate Image" is selected, the BOT will determine it's own prompt for image generation.
- The "Download Image" button prompts for a URL to use a specified image from a URL. The image will be downloaded then uploaded to wordpress and attached to the post.
- The "Find Image on Unsplash" button prompts to search Unsplash for an image to attach.  If no search terms are provided, the BOT will determine it's own search terms.
- The "Include YT Video" button prompts for a URL to a YouTube video.  The video will be embedded in the post.

### Ideas
- From the Ideas screen you can brainstorm ideas for a blog post.  You can use a vague concept and have the BOT a number of more concrete ideas to write about.
- You can also provide no concept and have the BOT generate a number of concepts and that same number of ideas to write about for each of those concepts
- You can manually add new as well as easily edit / delete existing ideas.  
- You can launch the write screen from a listed idea.  
- Ideas sharing a concept will be passed along with new requests for ideas to prevent duplicates as much as possible.

### Series
- The series screen provides another way of using ideas, grouped together by a common prompt.  
- It's very similar to Idea Concepts and may be merged or expanded to give it more clear purpose.

## Stable Diffusion
To use Stable Diffusion to generate images you'll need a functioning install of https://github.com/AUTOMATIC1111/stable-diffusion-webui with api enabled.
I am using https://hub.docker.com/r/universonic/stable-diffusion-webui with the following docker-compose.yml
```
services:
  sdweb:
    image: universonic/stable-diffusion-webui:latest
    ports:
     - YOUR_PORT_HERE:8080
    restart: unless-stopped
    volumes:
     - /LOCAL_DIR/extensions:/app/stable-diffusion-webui/extensions
     - /LOCAL_DIR/models:/app/stable-diffusion-webui/models
     - /LOCAL_DIR/outputs:/app/stable-diffusion-webui/outputs
     - /LOCAL_DIR/localizations:/app/stable-diffusion-webui/localizations
     - /LOCAL_DIR/entrypoint.sh:/app/entrypoint.sh
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```
#### Update /LOCAL_DIR to a path where you will store your models, extensions, outputs, etc.  You will need to create the directory structure.
#### Update YOUR_PORT_HERE to the port you want to use for the webui.

You'll need to get a checkpoint and VAE file and place them in the models directory.
1. The "default" 1.5 checkpoint: https://huggingface.co/runwayml/stable-diffusion-v1-5/blob/main/v1-5-pruned-emaonly.ckpt
2. This file will go in the "models/Stable-diffusion" directory.
3. AND this "VAE" file:
https://huggingface.co/stabilityai/sd-vae-ft-mse-original/blob/main/vae-ft-mse-840000-ema-pruned.ckpt
4. This file will go in the "models/VAE" directory.

You can find and download other checkpoints and VAE files here: https://huggingface.co/models?filter=stable-diffusion
But exercise caution as code can be embedded in models and you should only use models from trusted sources.

The Docker image as composed does not seem to work for me, so I've had to modify the entrypoint.sh to get it running.  We also need to modify it to add api access.
```
!/usr/bin/env bash
git -C /app/stable-diffusion-webui/ pull
/app/stable-diffusion-webui/webui.sh --api "$@"
```
Then run the docker-compose up -d

You will need to let the install complete til you get to a hang after successfully installing a bunch of dependencies, then restart the container.  It should work then.
I'm using an older nvidia Quadro P2000 GPU and have docker / cuda already up and running, consider having a proper setup a pre-req.