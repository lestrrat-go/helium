<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:value-of with toExpr -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="(2 to 4)[count(current()/*)]" separator="|"/>
         </t1>
         <t2>
            <t:value-of select="*[position() = last()]/concat(name(),string())" separator="|"/>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
