<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that it is allowed to have both select attribute and 
  				a sequence constructor for an xsl:message instruction. The result of 
  				evaluating them are concatenated to form a single sequence. -->

   <t:template match="/">
      <out>
         <t:message terminate="{/docs/terminate[1]}" select="'Please note that'">
        	   <t:for-each select="/docs/message">
  				         <t:value-of select="."/>
  			       </t:for-each>
        </t:message>
      </out>
  </t:template>
</t:transform>
