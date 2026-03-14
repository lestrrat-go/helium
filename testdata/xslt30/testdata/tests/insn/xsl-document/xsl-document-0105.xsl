<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:document to create a new document inside a variable
  				and get the text value stored in the new document using the variable. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="v">
		    <t:document>
			      <t:text> text inside document</t:text>
		    </t:document>
	  </t:variable>

   <t:template match="root">
		    <out>
         <t:value-of select="$v"/>
      </out>
	  </t:template>

   <t:template match="child"/>
</t:transform>
