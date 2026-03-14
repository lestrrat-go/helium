<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using xsl:document instruction inside an xsl:message. -->

   <t:variable name="v" as="document-node()">
		    <t:document>
	   		   <t:element name="smart">Smart element</t:element>
	   	</t:document>
	  </t:variable>

   <t:template match="docs">
   		<out>
         <t:message select="$v">
   				     <t:document>
   					       <t:element name="dummy">dummy element</t:element>
   				     </t:document>
   			   </t:message>
      </out>
	  </t:template>
</t:transform>
