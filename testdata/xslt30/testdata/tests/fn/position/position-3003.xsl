<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of fn:last() on sequence using RangeExpr -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="(1, 7 to 10)[last()]"/>
         </t1>
         <t2>
            <xslt:value-of select="(1, 7 to 10)[last() ge 4]"/>
         </t2>
         <t3>
            <xslt:value-of select="(1, 7 to 10)[last() lt 4] instance of empty-sequence()"/>
         </t3>
      </out>
   </xslt:template>
</xslt:transform>
