"""Exercise the actual render entry point without encoding media."""
import json,os,subprocess,sys,tempfile,unittest
from pathlib import Path

class CaptionContract(unittest.TestCase):
    def run_plan(self,subtitles):
        with tempfile.TemporaryDirectory() as temp:
            root=Path(temp);(root/'render').mkdir()
            scene={'id':'shot','duration':2,'clips':[{'file':'raw.webm','start':0,'length':2,'padStart':0,'padEnd':0}],
                   'overlay':'label.png','crop':None,'zoom':False,'audio':[]}
            plan={'duration':2,'subtitles':subtitles,'scenes':[scene],'transitionSeconds':.1,
                  'narrationTracks':[{'audio':'speech.wav','offset':.08,'duration':1.7}]}
            (root/'edit-plan.json').write_text(json.dumps(plan))
            script=Path(__file__).resolve().parents[1]/'render.py'
            subprocess.run([sys.executable,str(script),'--output',str(root)],env={**os.environ,'TUTORIAL_RENDER_DRY_RUN':'1'},check=True,capture_output=True)
            args=[json.loads(p.read_text()) for p in (root/'render').glob('*.args.json')]
            filters=[a[i+1] for a in args for i,x in enumerate(a) if x=='-filter_complex']
            return filters,list((root/'render').glob('*.ass')),args

    def test_subtitle_free_revision_has_no_caption_filter_or_speed_change(self):
        filters,ass,args=self.run_plan(False)
        self.assertFalse(ass)
        self.assertFalse(any('ass=filename' in f or 'subtitles=' in f for f in filters))
        self.assertFalse(any('atempo=' in f or 'asetrate=' in f for f in filters))
        self.assertFalse(any('-c:s' in a for a in args))
        self.assertTrue(any('adelay=80' in f and 'amix=' in f for f in filters))

    def test_existing_captioned_projects_keep_their_explicit_setting(self):
        filters,ass,_=self.run_plan(True)
        self.assertTrue(ass)
        self.assertTrue(any('ass=filename' in f for f in filters))

if __name__=='__main__':unittest.main()
