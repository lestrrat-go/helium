<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using xsl:message instruction inside xsl:element instructions. -->

   <t:template match="docs">
   		<out>
         <t:element name="E">
	   			     <t:message select="'My funny element'"/>
   			   </t:element>
      </out>
	  </t:template>
</t:transform>
