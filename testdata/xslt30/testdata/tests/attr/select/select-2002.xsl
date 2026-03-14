<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:value-of with document-node() and predicate index -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="current()/ancestor::document-node()[position() eq 1]"
                        separator="|"/>
         </t1>
         <t2>
            <t:value-of select="(/child::*/ancestor-or-self::document-node())[position() eq last()]"
                        separator="|"/>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
