import prom from "k6/x/prometheus";
import { sleep, check } from "k6";

export const options = {
    vus: 2,
    duration: "20s",
};

export default async function () {
    prom.init("http://prometheus:9090", 500);

    // ---------- SIMPLE QUERY ----------
    const samples = prom.query('up');
    console.log("UP:", samples);

    // ---------- RANGE ----------
    const now = Date.now();
    const range = prom.queryRange(
        'rate(node_cpu_seconds_total[1m])',
        now - 60000,
        now,
        5000
    );

    console.log("RANGE size:", range.length);

    // ---------- ASYNC ----------
    const asyncSamples = await prom.queryAsync('up');
    console.log("ASYNC:", asyncSamples.length);

    // ---------- WAIT FOR ----------
    const ready = prom.waitFor('up', 0.5, 5000);
    console.log("WAIT RESULT:", ready);

    // ---------- CHECK ----------
    check(null, {
        "node is up": () =>
            prom.check('up', (v) => v === 1),
    });

    sleep(1);
}