<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that for an xsl:comment instruction the text nodes
  				specified in the sequence constructor are concatenated with no separator.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
  		  <out>
         <t:comment>
  			       <t:text>b1</t:text>
  			       <t:text>b2</t:text>
  			       <t:text>b3</t:text>
  			       <t:text>b4</t:text>
  		     </t:comment>
      </out>
	  </t:template>

   <t:template match="node()">
  		  <t:apply-templates/>
	  </t:template>
</t:transform>
