<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using xsl:message instruction inside xsl:attribute instructions. -->

   <t:template match="docs">
   		<out>
         <t:element name="E">
   			      <t:attribute name="attr">
   				        <t:message>My funny attribute</t:message>
   				        <t:value-of select="'attrValue'"/>
   			      </t:attribute>
   			   </t:element>
      </out>
	  </t:template>
</t:transform>
