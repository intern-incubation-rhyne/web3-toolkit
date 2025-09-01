#!/usr/bin/env python3
"""
Liquidation Data Visualization Script
Creates monthly revenue charts from liquidation data
"""

import json
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime, timezone
import seaborn as sns
from pathlib import Path
import numpy as np

def load_data(file_path):
    """Load liquidation data from JSON file"""
    try:
        with open(file_path, 'r') as f:
            data = json.load(f)
        print(f"Loaded {len(data)} records from {file_path}")
        return data
    except FileNotFoundError:
        print(f"File not found: {file_path}")
        return []
    except json.JSONDecodeError:
        print(f"Invalid JSON in file: {file_path}")
        return []

def convert_timestamp_to_date(timestamp):
    """Convert Unix timestamp to datetime"""
    return datetime.fromtimestamp(timestamp, tz=timezone.utc)

def process_data(data):
    """Process data and create monthly aggregates"""
    if not data:
        return pd.DataFrame()
    
    # Convert to DataFrame
    df = pd.DataFrame(data)
    
    # Convert blockTimestamp to datetime
    df['date'] = df['blockTimestamp'].apply(convert_timestamp_to_date)
    
    # Extract month and year
    df['month_year'] = df['date'].dt.to_period('M')
    
    # Convert revenue to ETH (assuming revenue is in wei)
    df['revenue_eth'] = df['revenue'] / 1e18
    
    # Group by month and sum revenue
    monthly_revenue = df.groupby('month_year')['revenue_eth'].sum().reset_index()
    monthly_revenue['month_year'] = monthly_revenue['month_year'].astype(str)
    
    # Convert month_year back to datetime for plotting
    monthly_revenue['date'] = pd.to_datetime(monthly_revenue['month_year'])
    
    return monthly_revenue, df

def create_monthly_chart(monthly_data, output_dir="charts"):
    """Create monthly revenue column chart"""
    if monthly_data.empty:
        print("No data to plot")
        return
    
    # Set style - use default matplotlib style for compatibility
    plt.style.use('default')
    sns.set_theme(style="whitegrid")
    
    # Create figure
    fig, ax = plt.subplots(figsize=(16, 10))
    
    # Create bar chart with wider bars
    bars = ax.bar(monthly_data['date'], monthly_data['revenue_eth'], 
                  width=20, color='skyblue', alpha=0.7, edgecolor='navy', linewidth=0.5)
    
    # Customize chart
    ax.set_title('Monthly Liquidation Revenue (Jan 2025 - Present)', 
                fontsize=16, fontweight='bold', pad=20)
    ax.set_xlabel('Month', fontsize=12, fontweight='bold')
    ax.set_ylabel('Revenue (ETH)', fontsize=12, fontweight='bold')
    
    # Format x-axis
    ax.xaxis.set_major_locator(mdates.MonthLocator(interval=1))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%b %Y'))
    plt.xticks(rotation=45, ha='right')
    
    # Add value labels on bars
    for bar in bars:
        height = bar.get_height()
        if height > 0:
            ax.text(bar.get_x() + bar.get_width()/2., height + height*0.01,
                   f'{height:.4f}', ha='center', va='bottom', fontsize=9)
    
    # Add grid
    ax.grid(True, alpha=0.3, axis='y')
    
    # Adjust layout
    plt.tight_layout()
    
    # Create output directory
    Path(output_dir).mkdir(exist_ok=True)
    
    # Save chart
    output_path = Path(output_dir) / "monthly_revenue_chart.png"
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    print(f"Chart saved to: {output_path}")
    
    # Show chart
    plt.show()

def create_detailed_analysis(df):
    """Create detailed analysis of the data"""
    if df.empty:
        return
    
    print("\n=== DETAILED ANALYSIS ===")
    print(f"Total records: {len(df)}")
    print(f"Date range: {df['date'].min().strftime('%Y-%m-%d')} to {df['date'].max().strftime('%Y-%m-%d')}")
    print(f"Total revenue: {df['revenue_eth'].sum():.6f} ETH")
    print(f"Average revenue per liquidation: {df['revenue_eth'].mean():.6f} ETH")
    print(f"Median revenue per liquidation: {df['revenue_eth'].median():.6f} ETH")
    print(f"Max revenue: {df['revenue_eth'].max():.6f} ETH")
    print(f"Min revenue: {df['revenue_eth'].min():.6f} ETH")
    
    # Top 10 liquidations by revenue
    print("\n=== TOP 10 LIQUIDATIONS BY REVENUE ===")
    top_10 = df.nlargest(10, 'revenue_eth')[['date', 'transactionHash', 'revenue_eth']]
    for _, row in top_10.iterrows():
        print(f"{row['date'].strftime('%Y-%m-%d %H:%M')} | {row['transactionHash'][:10]}... | {row['revenue_eth']:.6f} ETH")

def create_additional_charts(df, output_dir="charts"):
    """Create additional visualization charts"""
    if df.empty:
        return
    
    # Create output directory
    Path(output_dir).mkdir(exist_ok=True)
    
    # 1. Daily revenue trend
    daily_revenue = df.groupby(df['date'].dt.date)['revenue_eth'].sum().reset_index()
    daily_revenue['date'] = pd.to_datetime(daily_revenue['date'])
    
    plt.figure(figsize=(14, 6))
    plt.plot(daily_revenue['date'], daily_revenue['revenue_eth'], marker='o', linewidth=2, markersize=4)
    plt.title('Daily Liquidation Revenue Trend', fontsize=14, fontweight='bold')
    plt.xlabel('Date', fontsize=12)
    plt.ylabel('Revenue (ETH)', fontsize=12)
    plt.grid(True, alpha=0.3)
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(Path(output_dir) / "daily_revenue_trend.png", dpi=300, bbox_inches='tight')
    plt.show()
    
    # 2. Revenue distribution histogram
    plt.figure(figsize=(12, 6))
    plt.hist(df['revenue_eth'], bins=50, alpha=0.7, color='lightcoral', edgecolor='black')
    plt.title('Revenue Distribution', fontsize=14, fontweight='bold')
    plt.xlabel('Revenue (ETH)', fontsize=12)
    plt.ylabel('Frequency', fontsize=12)
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(Path(output_dir) / "revenue_distribution.png", dpi=300, bbox_inches='tight')
    plt.show()

def main():
    """Main function"""
    print("=== LIQUIDATION DATA VISUALIZATION ===\n")
    data = load_data("data/mainnet_euler_logs_with_revenue.json")
    
    # Process data
    monthly_data, df = process_data(data)
    
    if monthly_data.empty:
        print("No valid data to process")
        return
    
    # Create visualizations
    print("\nCreating monthly revenue chart...")
    create_monthly_chart(monthly_data)
    
    # # Create additional charts
    # print("\nCreating additional charts...")
    # create_additional_charts(df)
    
    # # Print detailed analysis
    # create_detailed_analysis(df)
    
    print("\n=== VISUALIZATION COMPLETE ===")
    print("Charts have been saved to the 'charts' directory")

if __name__ == "__main__":
    main()
