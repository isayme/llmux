import React, { useState, useEffect } from 'react';
import { Table, Tag, Space, Select, DatePicker, Button, Modal, Typography, Card } from 'antd';
import { ReloadOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;
const { Text } = Typography;

interface RequestLog {
  id: number;
  request_id: string;
  timestamp: string;
  model: string;
  alias: string;
  method: string;
  path: string;
  client_ip: string;
  api_key_id: string;
  duration: number;
  status: string;
}

interface ProviderCall {
  id: number;
  provider_id: string;
  provider_type: string;
  model: string;
  response_code: number;
  duration: number;
  is_retry: boolean;
  error: string;
}

const LogsPage: React.FC = () => {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [modelFilter, setModelFilter] = useState<string>('');
  const [timeRange, setTimeRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);
  const [selectedLog, setSelectedLog] = useState<RequestLog | null>(null);
  const [providerCalls, setProviderCalls] = useState<ProviderCall[]>([]);
  const [detailModalVisible, setDetailModalVisible] = useState(false);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      });
      
      if (statusFilter) params.append('status', statusFilter);
      if (modelFilter) params.append('model', modelFilter);
      if (timeRange?.[0]) params.append('start_time', timeRange[0].toISOString());
      if (timeRange?.[1]) params.append('end_time', timeRange[1].toISOString());
      
      const response = await fetch(`/api/logs/requests?${params}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('master_key')}`,
        },
      });
      
      if (response.ok) {
        const data = await response.json();
        setLogs(data.data || []);
        setTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchProviderCalls = async (requestLogId: number) => {
    try {
      const response = await fetch(`/api/logs/requests/${requestLogId}/calls`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('master_key')}`,
        },
      });
      
      if (response.ok) {
        const data = await response.json();
        setProviderCalls(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch provider calls:', error);
    }
  };

  const handleViewDetail = async (record: RequestLog) => {
    setSelectedLog(record);
    await fetchProviderCalls(record.id);
    setDetailModalVisible(true);
  };

  const handleDelete = async () => {
    Modal.confirm({
      title: 'Confirm Delete',
      content: 'Are you sure you want to delete logs older than 7 days?',
      onOk: async () => {
        try {
          const response = await fetch('/api/logs/requests?days=7', {
            method: 'DELETE',
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('master_key')}`,
            },
          });
          
          if (response.ok) {
            fetchLogs();
          }
        } catch (error) {
          console.error('Failed to delete logs:', error);
        }
      },
    });
  };

  useEffect(() => {
    fetchLogs();
  }, [page, pageSize, statusFilter, modelFilter, timeRange]);

  const columns: ColumnsType<RequestLog> = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: 'Request ID',
      dataIndex: 'request_id',
      key: 'request_id',
      width: 200,
      render: (text: string) => <Text copyable>{text}</Text>,
    },
    {
      title: 'Model',
      dataIndex: 'model',
      key: 'model',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'success' ? 'green' : 'red'}>
          {status}
        </Tag>
      ),
    },
    {
      title: 'Duration',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration: number) => `${duration}ms`,
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record)}
          >
            View
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card title="Request Logs" extra={
        <Space>
          <RangePicker
            onChange={(dates) => setTimeRange(dates as [dayjs.Dayjs | null, dayjs.Dayjs | null])}
          />
          <Select
            placeholder="Status"
            allowClear
            style={{ width: 120 }}
            onChange={setStatusFilter}
            options={[
              { label: 'Success', value: 'success' },
              { label: 'Failed', value: 'failed' },
            ]}
          />
          <input
            placeholder="Model"
            value={modelFilter}
            onChange={(e) => setModelFilter(e.target.value)}
            style={{ padding: '4px 8px', borderRadius: 4, border: '1px solid #d9d9d9' }}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchLogs}>
            Refresh
          </Button>
          <Button icon={<DeleteOutlined />} onClick={handleDelete} danger>
            Clean Old
          </Button>
        </Space>
      }>
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      <Modal
        title="Request Details"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        width={800}
      >
        {selectedLog && (
          <div>
            <Card size="small" title="Request Info" style={{ marginBottom: 16 }}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div><strong>Request ID:</strong> {selectedLog.request_id}</div>
                <div><strong>Model:</strong> {selectedLog.model}</div>
                <div><strong>Method:</strong> {selectedLog.method}</div>
                <div><strong>Path:</strong> {selectedLog.path}</div>
                <div><strong>Client IP:</strong> {selectedLog.client_ip}</div>
                <div><strong>Duration:</strong> {selectedLog.duration}ms</div>
              </Space>
            </Card>
            
            <Card size="small" title="Provider Calls">
              <Table
                dataSource={providerCalls}
                rowKey="id"
                pagination={false}
                size="small"
                columns={[
                  { title: 'Provider', dataIndex: 'provider_id', key: 'provider_id' },
                  { title: 'Model', dataIndex: 'model', key: 'model' },
                  { 
                    title: 'Status', 
                    dataIndex: 'response_code', 
                    key: 'response_code',
                    render: (code: number) => (
                      <Tag color={code >= 200 && code < 300 ? 'green' : 'red'}>
                        {code}
                      </Tag>
                    ),
                  },
                  { title: 'Duration', dataIndex: 'duration', key: 'duration', render: (d: number) => `${d}ms` },
                  { 
                    title: 'Retry', 
                    dataIndex: 'is_retry', 
                    key: 'is_retry',
                    render: (isRetry: boolean) => isRetry ? <Tag color="orange">Yes</Tag> : 'No',
                  },
                ]}
              />
            </Card>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default LogsPage;
