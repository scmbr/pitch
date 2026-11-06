import { test } from 'node:test';
import assert from 'node:assert/strict';
import { compareSlides, boundedSeek } from '../src/lib/comparison.js';

test('comparison sums revisited slides and retains slides absent in one attempt', () => {
  assert.deepEqual(compareSlides(
    [{slideIndex:0,start:0,end:2},{slideIndex:1,start:2,end:5},{slideIndex:0,start:5,end:6}],
    [{slideIndex:0,start:0,end:2},{slideIndex:2,start:2,end:6}],
  ), [
    {slide:0,first:3,second:2,delta:-1},
    {slide:1,first:3,second:0,delta:-3},
    {slide:2,first:0,second:4,delta:4},
  ]);
});
test('shorter recording seeks clamp independently and empty timeline is allowed', () => {
  assert.equal(boundedSeek(10,3),2.999);
  assert.equal(boundedSeek(-1,3),0);
  assert.equal(boundedSeek(1,0),0);
  assert.deepEqual(compareSlides([],[]),[]);
});
