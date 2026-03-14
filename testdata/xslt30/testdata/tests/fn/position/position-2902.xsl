<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of fn:last() with sequence consisting of atomic values -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="(1, xs:boolean('true'), xs:gMonthDay('--10-17'), 2.343, xs:gYearMonth('2005-10'))[last()]"/>
         </t1>
         <t2>
            <xslt:value-of select="(1, xs:boolean('true'), xs:gMonthDay('--10-17'), 2.343, xs:gYearMonth('2005-10'))[last() gt 6] instance of empty-sequence()"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
