<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of fn:last() in predicate in conjunction with wildcard -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="*:inner[last()]"/>
         </t1>
         <t2>
            <xslt:value-of select="tests:*[last()]"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
