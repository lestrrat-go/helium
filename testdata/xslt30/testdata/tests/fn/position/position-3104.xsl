<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:tests="http://www.test.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs tests">
<!-- Purpose: Test of position with forward axes -->

   <xslt:variable name="var">
      <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <inner1>k1<cd>c1</cd>
         </inner1>
         <inner1>k2</inner1>
         <inner1>k3</inner1>
         <inner1>k4</inner1>
      </doc>
   </xslt:variable>

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:value-of select="count($var/child::*/descendant-or-self::*)"/>
         </t1>
         <t2>
            <xslt:value-of select="($var/child::*/descendant-or-self::*)[3]"/>
         </t2>
      </out>
   </xslt:template>
</xslt:transform>
