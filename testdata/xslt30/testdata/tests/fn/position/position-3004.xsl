<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of fn:count() on sequence using ForExpr -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="count(for $var in (2 to 10) return $var+1)"/>
         </t1>
         <t2>
            <xslt:value-of select="count(for $var in (2 to 10) return ($var, 1))"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
