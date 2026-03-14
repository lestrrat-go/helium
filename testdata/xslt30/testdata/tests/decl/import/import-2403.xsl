<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that is missing the REQUIRED href attribute in an xsl:import instruction. -->

   <t:import/>

   <t:template match="/">
	     <out>
         <t:element name="test">
	    	      <t:attribute name="status" select="'Invalid'"/>
	        </t:element>
      </out>
	  </t:template>
</t:transform>
