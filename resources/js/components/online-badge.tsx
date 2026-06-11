import {
    Badge,
    Tooltip,
} from 'antd';

import {
    DeviceStatus,
    useDeviceStatus,
} from '../hooks/useDeviceStatus';

type Props = {
    deviceId: number;
    initialStatus?: string;
};

const statusConfig: Record<DeviceStatus, { color: string; label: string }> = {
    online: { color: '#52c41a', label: 'Online' },
    offline: { color: '#ff4d4f', label: 'Offline' },
    no_session: { color: '#d9d9d9', label: 'Belum ada session' },
};

export default function DeviceStatusBadge({ deviceId, initialStatus }: Props) {
    const status = useDeviceStatus(deviceId, initialStatus);
    const { color, label } = statusConfig[status];

    return (
        <Tooltip title={label}>
            <Badge color={color} text={label} />
        </Tooltip>
    );
}