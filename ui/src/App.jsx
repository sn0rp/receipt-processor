import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

const API_URL = process.env.REACT_APP_API_URL || '';

function App() {
  const [receipts, setReceipts] = useState([]);
  const [newReceipt, setNewReceipt] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    fetchReceipts();
  }, []);

  const fetchReceipts = async () => {
    try {
      const response = await axios.get(`${API_URL}/receipts`);
      const receiptsWithPoints = await Promise.all(
        response.data.map(async (receipt) => {
          try {
            const pointsResponse = await axios.get(`${API_URL}/receipts/${receipt.id}/points`);
            return { ...receipt, points: pointsResponse.data.points };
          } catch (err) {
            console.error(`Error fetching points for receipt ${receipt.id}:`, err);
            return receipt;
          }
        })
      );
      setReceipts(receiptsWithPoints);
      setError('');
    } catch (err) {
      console.error('Error fetching receipts:', err);
      setError('Failed to fetch receipts: ' + (err.response?.data || err.message));
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    
    try {
      let receiptData;
      try {
        receiptData = JSON.parse(newReceipt);
      } catch (err) {
        setError('Invalid JSON format');
        return;
      }

      const response = await axios.post(`${API_URL}/receipts/process`, receiptData, {
        headers: {
          'Content-Type': 'application/json'
        }
      });

      if (response.data && response.data.id) {
        const newReceipt = {
          ...receiptData,
          id: response.data.id,
          points: response.data.points
        };
        setReceipts(prevReceipts => [...prevReceipts, newReceipt]);
        setNewReceipt('');
        setError('');
      } else {
        setError('Invalid response from server');
      }
    } catch (err) {
      console.error('Error processing receipt:', err);
      setError('Failed to process receipt: ' + (err.response?.data || err.message));
    }
  };

  return (
    <div className="App">
      <h1 className='page-title'>Receipt Processor</h1>
      {error && <div className="error">{error}</div>}
      
      <form onSubmit={handleSubmit}>
        <textarea
          value={newReceipt}
          onChange={(e) => setNewReceipt(e.target.value)}
          placeholder="Paste receipt JSON here..."
        />
        <button type="submit">Process Receipt</button>
      </form>

      <h2 className="section-header">Processed Receipts</h2>
      <div className="receipts-grid">
        {receipts.length === 0 ? (
          <p>No receipts processed yet.</p>
        ) : (
          receipts.map(receipt => (
            <div key={receipt.id} className="receipt">
              <h3>{receipt.retailer}</h3>
              <p>ID: {receipt.id}</p>
              <p>Date: {receipt.purchaseDate}</p>
              <p>Time: {receipt.purchaseTime}</p>
              <p>Total: ${receipt.total}</p>
              <p>Points: {receipt.points}</p>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

export default App; 