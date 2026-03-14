<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:comment to create two comment nodes.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:comment select="'Inside select attribute'"/>
         <t:comment>Inside the sequence constructor</t:comment>
      </out>
	  </t:template>
</t:transform>
