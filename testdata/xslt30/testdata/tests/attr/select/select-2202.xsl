<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:apply-templates with processing-instruction(NCName) with predicate -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="(self::*:doc/descendant-or-self::processing-instruction(pi))[last()]"
                        separator="|"/>
         </t1>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
