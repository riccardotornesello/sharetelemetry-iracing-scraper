import os
import json
import time
from google.cloud import pubsub_v1

from conf import PROJECT_ID


# Setup the environment variables
os.environ["PUBSUB_EMULATOR_HOST"] = "localhost:8085"

# Define the payload to be sent
DATA = {
    "api-req": [
        # {"endpoint": "/data/results/get", "params": {"subsession_id": "81891896", "include_licenses": "false"}},
        {"endpoint": "/data/league/season_sessions", "params": {"league_id": "4403", "season_id": "0"}},
    ]
}


def publish_json_message(publisher, topic_path, data_dict):
    try:
        # 1. Serialization: Dict -> JSON String
        json_str = json.dumps(data_dict)
        
        # 2. Encoding: JSON String -> Bytes
        data_bytes = json_str.encode("utf-8")

        # 3. Publishing
        future = publisher.publish(
            topic_path,
            data_bytes,
            origin="python-script", # Example of custom attribute
            type="test-element" # Example of custom attribute
        )
        
        # 4. Waiting for confirmation (Message ID)
        # .result() blocks the script until the server confirms receipt
        message_id = future.result()
        
        print(f"✅ Message published! ID: {message_id}")
        print(f"   Payload: {json_str}")
        
    except Exception as e:
        print(f"❌ Error during publishing: {e}")


if __name__ == "__main__":
    publisher = pubsub_v1.PublisherClient()

    for topic_id, data in DATA.items():
        topic_path = publisher.topic_path(PROJECT_ID, topic_id)

        print(f"📡 Sending data to: {topic_path}\n")
    
        for el in data:
            publish_json_message(publisher, topic_path, el)
            time.sleep(1)

        print("\n🏁 Sending completed.")
