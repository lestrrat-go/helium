<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Function is invoked with an XML document that contains just a 
  				single empty element tag that has no attributes. -->

   <t:template match="/">
	 	   <out>
         <t:value-of select="root( doc )"/>
      </out>
	  </t:template>
</t:transform>
