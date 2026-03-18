import prom from 'k6/x/prometheus';
import { sleep } from 'k6';

export const options = {
    vus: 5,
    duration: '20s',
};

const client = new prom.NewClient({
    address: 'http://prometheus:9090',
});

export default function () {
    const res = client.Query('avg(rate(node_cpu_seconds_total{mode="idle"}[1m]))');
    const value = res.asNumber();

    console.log(`CPU idle: ${value}`);

    const overloaded = client.EvaluateThreshold(
        '1 - avg(rate(node_cpu_seconds_total{mode="idle"}[1m]))',
        0.2
    );

    if (overloaded) {
        console.log('🔥 SYSTEM UNDER LOAD — slowing down');
        sleep(2);
    } else {
        sleep(0.5);
    }
}