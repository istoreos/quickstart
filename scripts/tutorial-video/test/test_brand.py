import array,math,sys,unittest
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from brand_pronunciation import compact_pcm

class BrandSilence(unittest.TestCase):
    def test_only_long_silence_is_removed_and_voice_samples_are_exact(self):
        rate=24000;voice=array.array('h',(round(12000*math.sin(i*.2)) for i in range(9600)))
        raw=voice+array.array('h',[0]*12000)+voice
        result,cuts=compact_pcm(raw,rate,[(.4,.9)])
        self.assertEqual(result[:9600],voice);self.assertEqual(result[-9600:],voice)
        self.assertAlmostEqual((len(result)-19200)/rate,.06,places=3)
        self.assertTrue(all(all(x==0 for x in raw[a:b]) for a,b in cuts))
    def test_short_pause_is_retained(self):
        raw=array.array('h',[0]*1920)
        result,cuts=compact_pcm(raw,24000,[(0,.08)])
        self.assertEqual(result,raw);self.assertFalse(cuts)
    def test_voice_inside_asr_gap_is_never_cut(self):
        raw=array.array('h',[2000]*24000)
        result,cuts=compact_pcm(raw,24000,[(.1,.9)])
        self.assertEqual(result,raw);self.assertFalse(cuts)

if __name__=='__main__':unittest.main()
