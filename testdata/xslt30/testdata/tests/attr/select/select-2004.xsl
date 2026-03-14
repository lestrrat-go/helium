<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:value-of with attribute() and predicate index -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="self::*/attribute()[name() eq 'att2'][last()]" separator="|"/>
         </t1>
         <t2>
            <t:value-of select="(/child::*/ancestor-or-self::*/attribute())[name() eq 'att2' and last()]"
                        separator="|"/>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
