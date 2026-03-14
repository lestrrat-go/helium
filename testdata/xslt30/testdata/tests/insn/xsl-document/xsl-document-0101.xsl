<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test case that uses <xsl:document> instruction to create an 
  				empty new document node (sequence constructor does not contain any instruction).-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="root">
		    <out>
         <t:document/>
      </out>
	  </t:template>

   <t:template match="child"/>
</t:transform>
