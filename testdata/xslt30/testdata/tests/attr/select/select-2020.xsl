<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with document-node() and predicate index -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:for-each select="current()/ancestor::document-node()[position() eq last()][*]">
	  				        <t:value-of select="."/>
	  				        <t:text>|</t:text>
	  			      </t:for-each>
         </t1>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
