<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:value-of to create text node while the value of 
  				the new text node is defined by the sequence constructor rather than the 
  				select attribute. 
  				Test case uses Literal text in the sequence constructor of an xsl:value-of 
  				instruction with a default separator.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:value-of> What happens if there is no select attribute?</t:value-of>
      </out>
	  </t:template>
</t:transform>
