<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of fn:position() in predicate in conjunction with wildcard -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="*:inner[position()=4]"/>
         </t1>
         <t2>
            <xslt:value-of select="*:inner[4]"/>
         </t2>
         <t3>
            <xslt:value-of select="tests:*[position()=1]"/>
         </t3>
         <t4>
            <xslt:value-of select="tests:*[1]"/>
         </t4>
      </out>
   </xslt:template>
</xslt:transform>
