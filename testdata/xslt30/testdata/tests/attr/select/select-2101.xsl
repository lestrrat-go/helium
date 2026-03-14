<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:test="http://www.test.org/Transform"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="test">
<!-- Purpose: Test of select in xsl:value-of with *:NCName and NCName:* -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="self::*:doc/(descendant-or-self::*:num3 | descendant-or-self::*:num1)"
                        separator="|"/>
         </t1>
         <t2>
            <t:value-of select="self::*:doc/(descendant-or-self::test:*)" separator="|"/>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
