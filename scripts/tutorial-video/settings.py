"""Private workstation settings, deliberately separate from versioned presets."""
import json
import os
from pathlib import Path


def load_settings():
    path=Path(os.environ.get('TUTORIAL_CONFIG',Path(__file__).parent/'.local/config.json'))
    if not path.is_file():
        raise SystemExit('Create a private config from config.example.json; set TUTORIAL_CONFIG if needed.')
    return json.loads(path.read_text(encoding='utf-8'))


def windows_environment(settings):
    values={'TUTORIAL_SERVER_ROOT':settings['serverRoot'],'TUTORIAL_OUTPUT':settings['output'],
            'TUTORIAL_FFMPEG':settings['ffmpeg'],'TUTORIAL_FFPROBE':settings['ffprobe'],
            'TUTORIAL_TTS_ENDPOINT':settings['ttsEndpoint'],'TUTORIAL_VOICE_REFERENCE':settings['voiceReference']}
    if settings.get('parentOutput'):
        values['TUTORIAL_PARENT_OUTPUT']=settings['parentOutput']
    for key,env in [('speechDriver','TUTORIAL_SPEECH_DRIVER'),('speechPython','TUTORIAL_SPEECH_PYTHON'),('speechStyle','TUTORIAL_SPEECH_STYLE')]:
        if settings.get(key):values[env]=settings[key]
    return ''.join("$env:"+key+"='"+value.replace("'","''")+"'; " for key,value in values.items())
