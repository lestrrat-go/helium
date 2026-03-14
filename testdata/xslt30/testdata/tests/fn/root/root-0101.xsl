<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Function is invoked with a document node -->

   <t:template match="document-node()">
	 	   <out>
         <t:value-of select="root(.)"/>
      </out>
	  </t:template>
</t:transform>
