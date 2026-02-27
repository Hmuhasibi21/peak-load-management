import http from 'k6/http';
import { sleep, check } from 'k6';

export const options = {
  vus: 10, // 10 User simulasi
  duration: '30s', // durasi 30 detik
};

export default function () {
  const res = http.get('http://localhost:8080/ping');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response text': (r) => r.body.includes('Hello dari API ElasticSix'),
  });
  sleep(1);
}