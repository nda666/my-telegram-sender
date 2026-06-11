import {
    useEffect,
    useState,
} from 'react';

export type DeviceStatus = 'online' | 'offline' | 'no_session';

export function useDeviceStatus(deviceId: number, initialStatus?: string): DeviceStatus {
    const [status, setStatus] = useState<DeviceStatus>(
        (initialStatus as DeviceStatus) ?? 'offline'
    );

    useEffect(() => {
        const es = new EventSource(`/devices/${deviceId}/status/stream`);

        es.onmessage = (e) => {
            const s = e.data.trim() as DeviceStatus;
            setStatus(s);
        };

        es.onerror = () => {
            // reconnect otomatis oleh browser, tidak perlu handle manual
        };

        return () => es.close();
    }, [deviceId]);

    return status;
}