<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests the use of character references to the control characters #x1 through 
  				#x1F in the input document, no error should be raised if the xml declaration 
  				of the input document has verion=1.1 and the characters are not copied to 
  				the output document. -->

   <t:output method="xml" version="1.1"/>

   <t:template match="/">
	     <out>
         <t:for-each select="doc/child::*">
			         <t:value-of select="."/>
		       </t:for-each>
      </out>
  </t:template>
</t:transform>
